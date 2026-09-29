package repository

import (
	"context"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// Select and reserve in one operation: stale read/then-acquire loops cannot
// guarantee projected utilization when many requests arrive simultaneously.
var independentAcquireScript = redis.NewScript(`
redis.replicate_commands()
local t=redis.call('TIME'); local now=tonumber(t[1]); local ttl=tonumber(ARGV[1])
local best=0; local priority=nil; local numerator=0; local denominator=1; local ties=0
for i=1,tonumber(ARGV[3]) do
 local base=3+(i-1)*6
 local p=tonumber(ARGV[base+2]); local factor=tonumber(ARGV[base+3]); local cap=tonumber(ARGV[base+4])
 local slot=KEYS[(i-1)*3+1]; local live=KEYS[(i-1)*3+2]; local paused=KEYS[(i-1)*3+3]
 redis.call('ZREMRANGEBYSCORE',slot,'-inf',now-ttl)
 redis.call('ZREMRANGEBYSCORE',live,'-inf',now-60)
 local n=redis.call('ZCARD',slot)+redis.call('ZCARD',live)
 local blocked=ARGV[base+6]=='0' and redis.call('GET',paused)==ARGV[base+5]
 if not blocked and (cap<=0 or n<cap) then
  if best==0 or p<priority or (p==priority and (n+1)*denominator<numerator*factor) then
   best=i;priority=p;numerator=n+1;denominator=factor;ties=1
  elseif p==priority and (n+1)*denominator==numerator*factor then
   ties=ties+1; if math.random(ties)==1 then best=i end
  end
 end
end
if best==0 then return 0 end
local slot=KEYS[(best-1)*3+1]
redis.call('ZADD',slot,now,ARGV[2]);redis.call('EXPIRE',slot,ttl)
return tonumber(ARGV[4+(best-1)*6])
`)

func (c *concurrencyCache) AcquireScheduledAccount(ctx context.Context, candidates []service.SchedulingCandidate, requestID string) (int64, error) {
	if len(candidates) == 0 {
		return 0, nil
	}
	keys := make([]string, 0, len(candidates)*3)
	args := []any{c.slotTTLSeconds, requestID, len(candidates)}
	for _, a := range candidates {
		id := strconv.FormatInt(a.ID, 10)
		keys = append(keys, accountSlotKeyPrefix+id, liveAccountSlotKeyPrefix+id, slowTTFTKeys(a.ID, a.PauseGeneration)[2])
		ignore := 0
		if a.IgnoreSlowTTFT {
			ignore = 1
		}
		args = append(args, a.ID, a.Priority, a.LoadFactor, a.Concurrency, a.PauseGeneration, ignore)
	}
	id, err := independentAcquireScript.Run(ctx, c.rdb, keys, args...).Int64()
	if err == nil && id > 0 {
		c.refreshAccountActiveIndex(ctx, id)
	}
	return id, err
}
