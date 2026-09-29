package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const slowTTFTPausePrefix = "slow_ttft:pause:"

var slowTTFTObserveScript = redis.NewScript(`
redis.replicate_commands()
local t=redis.call('TIME');local now=tonumber(t[1])*1000+math.floor(tonumber(t[2])/1000)
if redis.call('GET',KEYS[3])==ARGV[1] then return {tonumber(redis.call('HGET',KEYS[1],'until') or '0'),tonumber(redis.call('HGET',KEYS[1],'reason') or '0')} end
local window=tonumber(ARGV[3])*1000
local olduntil=tonumber(redis.call('HGET',KEYS[1],'until') or '0')
if olduntil>0 and olduntil<=now then redis.call('DEL',KEYS[1],KEYS[2],KEYS[4]) end
if redis.call('HGET',KEYS[1],'generation')~=ARGV[1] then redis.call('DEL',KEYS[1],KEYS[2],KEYS[3],KEYS[4]);redis.call('HSET',KEYS[1],'generation',ARGV[1]) end
redis.call('ZREMRANGEBYSCORE',KEYS[4],'-inf',now-math.max(window,60000))
if redis.call('ZADD',KEYS[4],'NX',now,ARGV[2])==0 then return {0,0} end
redis.call('PEXPIRE',KEYS[4],math.max(window,60000))
redis.call('ZREMRANGEBYSCORE',KEYS[2],'-inf',now-window)
local n=0
if ARGV[6]=='1' then
 n=redis.call('HINCRBY',KEYS[1],'consecutive',1);redis.call('ZADD',KEYS[2],now,ARGV[2])
else redis.call('HSET',KEYS[1],'consecutive',0) end
local reason=0
if n>=tonumber(ARGV[4]) then reason=1 elseif redis.call('ZCARD',KEYS[2])>=tonumber(ARGV[5]) then reason=2 end
local untilms=0
if reason>0 then
 untilms=now+tonumber(ARGV[7])*1000;redis.call('HSET',KEYS[1],'until',untilms,'reason',reason)
 redis.call('SET',KEYS[3],ARGV[1],'PX',tonumber(ARGV[7])*1000)
end
redis.call('EXPIRE',KEYS[1],math.max(tonumber(ARGV[7]),tonumber(ARGV[3]))+86400)
redis.call('EXPIRE',KEYS[2],math.max(tonumber(ARGV[7]),tonumber(ARGV[3]))+86400)
return {untilms,reason}
`)

// Epoch-scoped keys prevent old in-flight callbacks from resetting newer
// counters. Superseded epochs expire naturally; clearing requires no Redis write.
func slowTTFTKeys(id int64, generation string) []string {
	s := strconv.FormatInt(id, 10) + ":" + generation
	return []string{"slow_ttft:state:" + s, "slow_ttft:window:" + s, slowTTFTPausePrefix + s, "slow_ttft:seen:" + s}
}
func (c *tempUnschedCache) ObserveSlowTTFT(ctx context.Context, id int64, cfg service.SlowTTFTConfig, attempt string, slow bool) (service.SlowTTFTObservation, error) {
	s := 0
	if slow {
		s = 1
	}
	v, err := slowTTFTObserveScript.Run(ctx, c.rdb, slowTTFTKeys(id, cfg.Generation), cfg.Generation, attempt, cfg.WindowSeconds, cfg.ConsecutiveCount, cfg.WindowCount, s, cfg.PauseSeconds).Int64Slice()
	if err != nil {
		return service.SlowTTFTObservation{}, err
	}
	out := service.SlowTTFTObservation{}
	if len(v) == 2 && v[1] > 0 {
		out.Tripped = true
		out.Until = time.UnixMilli(v[0])
		out.Reason = "window"
		if v[1] == 1 {
			out.Reason = "consecutive"
		}
	}
	return out, nil
}
func (r *accountRepository) SetSlowTTFTPause(ctx context.Context, id int64, cfg service.SlowTTFTConfig, until time.Time, reason string) (bool, error) {
	res, err := r.sql.ExecContext(ctx, `WITH changed AS (UPDATE accounts SET slow_ttft_until=$2,slow_ttft_reason=$3,updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL AND extra->'slow_ttft_protection'->>'generation'=$4 AND extra->'slow_ttft_protection'->>'enabled'='true' AND (slow_ttft_until IS NULL OR slow_ttft_until<=NOW()) RETURNING id) INSERT INTO scheduler_outbox(event_type,account_id) SELECT $5,id FROM changed`, id, until, reason, cfg.Generation, service.SchedulerOutboxEventAccountChanged)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if n > 0 {
		r.syncSchedulerAccountSnapshot(ctx, id)
	}
	return n > 0, err
}
func (r *accountRepository) ClearSlowTTFTPause(ctx context.Context, id int64) error {
	// Durable invalidation is committed with the clear, even when Redis is down.
	res, err := r.sql.ExecContext(ctx, `WITH changed AS (UPDATE accounts SET slow_ttft_until=NULL,slow_ttft_reason='',extra=CASE WHEN extra ? 'slow_ttft_protection' THEN jsonb_set(extra,'{slow_ttft_protection,generation}',to_jsonb(md5(random()::text || clock_timestamp()::text))) ELSE extra END,updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL RETURNING id) INSERT INTO scheduler_outbox(event_type,account_id) SELECT $2,id FROM changed`, id, service.SchedulerOutboxEventAccountChanged)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return service.ErrAccountNotFound
	}
	r.syncSchedulerAccountSnapshot(ctx, id)
	return nil
}
