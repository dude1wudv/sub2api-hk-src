package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type charityRepository struct {
	db *sql.DB
}

// NewCharityRepository 创建公益池 PostgreSQL 仓储。
func NewCharityRepository(sqlDB *sql.DB) service.CharityRepository {
	return &charityRepository{db: sqlDB}
}

func (r *charityRepository) GetSettings(ctx context.Context) (*service.CharityPoolSettings, error) {
	const query = `
		SELECT
			enabled,
			daily_budget_usd,
			max_concurrency,
			per_ip_daily_limit_usd,
			per_ip_max_concurrency,
			per_ip_rpm_limit,
			auto_rotate_daily,
			rotation_timezone,
			site_title,
			site_notice,
			host_allowlist,
			system_user_id,
			created_at,
			updated_at
		FROM charity_pool_settings
		WHERE id = 1
	`
	s := &service.CharityPoolSettings{}
	var hosts pq.StringArray
	var systemUserID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query).Scan(
		&s.Enabled,
		&s.DailyBudgetUSD,
		&s.MaxConcurrency,
		&s.PerIPDailyLimitUSD,
		&s.PerIPMaxConcurrency,
		&s.PerIPRPMLimit,
		&s.AutoRotateDaily,
		&s.RotationTimezone,
		&s.SiteTitle,
		&s.SiteNotice,
		&hosts,
		&systemUserID,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return &service.CharityPoolSettings{
			Enabled:          false,
			DailyBudgetUSD:   service.CharityDefaultDailyBudgetUSD,
			MaxConcurrency:   service.CharityDefaultMaxConcurrency,
			AutoRotateDaily:  true,
			RotationTimezone: service.CharityDefaultTimezone,
			SiteTitle:        "XJTU Free AI",
			HostAllowlist:    []string{"free.xjtu.shop"},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get charity settings: %w", err)
	}
	s.HostAllowlist = []string(hosts)
	if systemUserID.Valid {
		v := systemUserID.Int64
		s.SystemUserID = &v
	}
	return s, nil
}

func (r *charityRepository) UpdateSettings(ctx context.Context, settings *service.CharityPoolSettings) (*service.CharityPoolSettings, error) {
	const query = `
		INSERT INTO charity_pool_settings (
			id,
			enabled,
			daily_budget_usd,
			max_concurrency,
			per_ip_daily_limit_usd,
			per_ip_max_concurrency,
			per_ip_rpm_limit,
			auto_rotate_daily,
			rotation_timezone,
			site_title,
			site_notice,
			host_allowlist,
			updated_at
		) VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			daily_budget_usd = EXCLUDED.daily_budget_usd,
			max_concurrency = EXCLUDED.max_concurrency,
			per_ip_daily_limit_usd = EXCLUDED.per_ip_daily_limit_usd,
			per_ip_max_concurrency = EXCLUDED.per_ip_max_concurrency,
			per_ip_rpm_limit = EXCLUDED.per_ip_rpm_limit,
			auto_rotate_daily = EXCLUDED.auto_rotate_daily,
			rotation_timezone = EXCLUDED.rotation_timezone,
			site_title = EXCLUDED.site_title,
			site_notice = EXCLUDED.site_notice,
			host_allowlist = EXCLUDED.host_allowlist,
			updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query,
		settings.Enabled,
		settings.DailyBudgetUSD,
		settings.MaxConcurrency,
		settings.PerIPDailyLimitUSD,
		settings.PerIPMaxConcurrency,
		settings.PerIPRPMLimit,
		settings.AutoRotateDaily,
		settings.RotationTimezone,
		settings.SiteTitle,
		settings.SiteNotice,
		pq.Array(settings.HostAllowlist),
	)
	if err != nil {
		return nil, fmt.Errorf("update charity settings: %w", err)
	}
	return r.GetSettings(ctx)
}

func (r *charityRepository) SetSystemUserID(ctx context.Context, userID int64) error {
	const query = `
		INSERT INTO charity_pool_settings (id, system_user_id, updated_at)
		VALUES (1, $1, NOW())
		ON CONFLICT (id) DO UPDATE SET
			system_user_id = EXCLUDED.system_user_id,
			updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("set charity system user id: %w", err)
	}
	return nil
}

const charityGroupSelectSQL = `
	SELECT
		cg.id,
		cg.group_id,
		cg.display_name,
		cg.description,
		cg.enabled,
		cg.daily_limit_usd,
		cg.max_concurrency,
		cg.sort_order,
		cg.active_api_key_id,
		COALESCE(ak.key, '') AS active_api_key,
		cg.last_rotated_date,
		cg.last_rotated_at,
		cg.created_at,
		cg.updated_at,
		COALESCE(g.name, '') AS group_name,
		COALESCE(g.platform, '') AS group_platform,
		COALESCE(g.status, '') AS group_status,
		COALESCE((
			SELECT COUNT(*)
			FROM account_groups ag
			JOIN accounts a ON a.id = ag.account_id AND a.deleted_at IS NULL
			WHERE ag.group_id = cg.group_id
		), 0) AS account_count,
		COALESCE((
			SELECT COUNT(*)
			FROM account_groups ag
			JOIN accounts a ON a.id = ag.account_id AND a.deleted_at IS NULL
			WHERE ag.group_id = cg.group_id AND a.status = 'active' AND a.schedulable = TRUE
		), 0) AS active_account_count
	FROM charity_groups cg
	JOIN groups g ON g.id = cg.group_id AND g.deleted_at IS NULL
	LEFT JOIN api_keys ak ON ak.id = cg.active_api_key_id AND ak.deleted_at IS NULL
`

func scanCharityGroup(scanner interface {
	Scan(dest ...any) error
}) (*service.CharityGroup, error) {
	g := &service.CharityGroup{}
	var dailyLimit sql.NullFloat64
	var activeKeyID sql.NullInt64
	var lastRotatedAt sql.NullTime
	err := scanner.Scan(
		&g.ID,
		&g.GroupID,
		&g.DisplayName,
		&g.Description,
		&g.Enabled,
		&dailyLimit,
		&g.MaxConcurrency,
		&g.SortOrder,
		&activeKeyID,
		&g.ActiveAPIKey,
		&g.LastRotatedDate,
		&lastRotatedAt,
		&g.CreatedAt,
		&g.UpdatedAt,
		&g.GroupName,
		&g.GroupPlatform,
		&g.GroupStatus,
		&g.AccountCount,
		&g.ActiveAccountCount,
	)
	if err != nil {
		return nil, err
	}
	if dailyLimit.Valid && dailyLimit.Float64 > 0 {
		v := dailyLimit.Float64
		g.DailyLimitUSD = &v
	}
	if activeKeyID.Valid {
		v := activeKeyID.Int64
		g.ActiveAPIKeyID = &v
	}
	if lastRotatedAt.Valid {
		t := lastRotatedAt.Time
		g.LastRotatedAt = &t
	}
	return g, nil
}

func (r *charityRepository) ListGroups(ctx context.Context) ([]service.CharityGroup, error) {
	rows, err := r.db.QueryContext(ctx, charityGroupSelectSQL+` ORDER BY cg.sort_order ASC, cg.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list charity groups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []service.CharityGroup
	for rows.Next() {
		item, scanErr := scanCharityGroup(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan charity group: %w", scanErr)
		}
		out = append(out, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate charity groups: %w", err)
	}
	return out, nil
}

func (r *charityRepository) GetGroupByID(ctx context.Context, id int64) (*service.CharityGroup, error) {
	row := r.db.QueryRowContext(ctx, charityGroupSelectSQL+` WHERE cg.id = $1`, id)
	g, err := scanCharityGroup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrCharityGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get charity group by id: %w", err)
	}
	return g, nil
}

func (r *charityRepository) GetGroupByGroupID(ctx context.Context, groupID int64) (*service.CharityGroup, error) {
	row := r.db.QueryRowContext(ctx, charityGroupSelectSQL+` WHERE cg.group_id = $1`, groupID)
	g, err := scanCharityGroup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrCharityGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get charity group by group_id: %w", err)
	}
	return g, nil
}

func (r *charityRepository) GetGroupByActiveAPIKeyID(ctx context.Context, apiKeyID int64) (*service.CharityGroup, error) {
	row := r.db.QueryRowContext(ctx, charityGroupSelectSQL+` WHERE cg.active_api_key_id = $1`, apiKeyID)
	g, err := scanCharityGroup(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrCharityGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get charity group by active_api_key_id: %w", err)
	}
	return g, nil
}

func (r *charityRepository) CreateGroup(ctx context.Context, g *service.CharityGroup) (*service.CharityGroup, error) {
	const query = `
		INSERT INTO charity_groups (
			group_id,
			display_name,
			description,
			enabled,
			daily_limit_usd,
			max_concurrency,
			sort_order,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id
	`
	var dailyLimit any
	if g.DailyLimitUSD != nil && *g.DailyLimitUSD > 0 {
		dailyLimit = *g.DailyLimitUSD
	}
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		g.GroupID,
		strings.TrimSpace(g.DisplayName),
		strings.TrimSpace(g.Description),
		g.Enabled,
		dailyLimit,
		g.MaxConcurrency,
		g.SortOrder,
	).Scan(&id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, service.ErrCharityGroupAlreadyExists
		}
		return nil, fmt.Errorf("create charity group: %w", err)
	}
	return r.GetGroupByID(ctx, id)
}

func (r *charityRepository) UpdateGroup(ctx context.Context, g *service.CharityGroup) (*service.CharityGroup, error) {
	const query = `
		UPDATE charity_groups
		SET
			display_name = $1,
			description = $2,
			enabled = $3,
			daily_limit_usd = $4,
			max_concurrency = $5,
			sort_order = $6,
			updated_at = NOW()
		WHERE id = $7
	`
	var dailyLimit any
	if g.DailyLimitUSD != nil && *g.DailyLimitUSD > 0 {
		dailyLimit = *g.DailyLimitUSD
	}
	res, err := r.db.ExecContext(ctx, query,
		strings.TrimSpace(g.DisplayName),
		strings.TrimSpace(g.Description),
		g.Enabled,
		dailyLimit,
		g.MaxConcurrency,
		g.SortOrder,
		g.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update charity group: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, service.ErrCharityGroupNotFound
	}
	return r.GetGroupByID(ctx, g.ID)
}

func (r *charityRepository) DeleteGroup(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM charity_groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete charity group: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrCharityGroupNotFound
	}
	return nil
}

func (r *charityRepository) SetActiveAPIKey(ctx context.Context, id int64, apiKeyID *int64, rotatedDate string, rotatedAt time.Time) error {
	const query = `
		UPDATE charity_groups
		SET
			active_api_key_id = $1,
			last_rotated_date = $2,
			last_rotated_at = $3,
			updated_at = NOW()
		WHERE id = $4
	`
	res, err := r.db.ExecContext(ctx, query, apiKeyID, rotatedDate, rotatedAt, id)
	if err != nil {
		return fmt.Errorf("set charity active api key: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrCharityGroupNotFound
	}
	return nil
}

func (r *charityRepository) GetDailyGroupUsage(ctx context.Context, usageDate string) ([]service.CharityGroupDailyUsage, error) {
	const query = `
		SELECT
			to_char(usage_date, 'YYYY-MM-DD'),
			group_id,
			total_cost_usd,
			request_count,
			total_tokens
		FROM charity_daily_group_usage
		WHERE usage_date = $1::date
	`
	rows, err := r.db.QueryContext(ctx, query, usageDate)
	if err != nil {
		return nil, fmt.Errorf("get charity daily group usage: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []service.CharityGroupDailyUsage
	for rows.Next() {
		var item service.CharityGroupDailyUsage
		if err := rows.Scan(&item.UsageDate, &item.GroupID, &item.TotalCostUSD, &item.RequestCount, &item.TotalTokens); err != nil {
			return nil, fmt.Errorf("scan charity daily group usage: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *charityRepository) GetDailyIPUsage(ctx context.Context, usageDate string, clientIP string) (*service.CharityIPDailyUsage, error) {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return &service.CharityIPDailyUsage{UsageDate: usageDate}, nil
	}
	const query = `
		SELECT
			to_char(usage_date, 'YYYY-MM-DD'),
			client_ip,
			total_cost_usd,
			request_count
		FROM charity_daily_ip_usage
		WHERE usage_date = $1::date AND client_ip = $2
	`
	item := &service.CharityIPDailyUsage{}
	err := r.db.QueryRowContext(ctx, query, usageDate, clientIP).Scan(
		&item.UsageDate,
		&item.ClientIP,
		&item.TotalCostUSD,
		&item.RequestCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return &service.CharityIPDailyUsage{UsageDate: usageDate, ClientIP: clientIP}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get charity daily ip usage: %w", err)
	}
	return item, nil
}

func (r *charityRepository) GetDailyTrend(ctx context.Context, days int) ([]service.CharityDailyTrendPoint, error) {
	if days <= 0 {
		days = 14
	}
	if days > 90 {
		days = 90
	}
	const query = `
		SELECT
			to_char(usage_date, 'YYYY-MM-DD') AS usage_date,
			COALESCE(SUM(total_cost_usd), 0) AS total_cost_usd,
			COALESCE(SUM(request_count), 0) AS request_count,
			COALESCE(SUM(total_tokens), 0) AS total_tokens
		FROM charity_daily_group_usage
		WHERE usage_date >= (CURRENT_DATE - ($1::int - 1))
		GROUP BY usage_date
		ORDER BY usage_date DESC
	`
	rows, err := r.db.QueryContext(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("get charity daily trend: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []service.CharityDailyTrendPoint
	for rows.Next() {
		var p service.CharityDailyTrendPoint
		if err := rows.Scan(&p.UsageDate, &p.TotalCostUSD, &p.RequestCount, &p.TotalTokens); err != nil {
			return nil, fmt.Errorf("scan charity daily trend: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
