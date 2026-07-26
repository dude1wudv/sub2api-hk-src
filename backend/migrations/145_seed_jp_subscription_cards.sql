-- Seed JP subscription card plans and redeem codes.
-- Codes are generated at migration runtime so usable redeem codes are not stored in git.

-- Groups backing subscription quota enforcement.
INSERT INTO groups (
    name,
    description,
    platform,
    subscription_type,
    daily_limit_usd,
    weekly_limit_usd,
    monthly_limit_usd,
    default_validity_days,
    status,
    rate_multiplier,
    sort_order,
    created_at,
    updated_at
)
SELECT
    'JP 100USD Daily Trial',
    'Trial subscription group: 100 USD daily quota for 1 day.',
    'openai',
    'subscription',
    100.00000000,
    NULL,
    NULL,
    1,
    'active',
    1.0,
    100,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM groups WHERE name = 'JP 100USD Daily Trial' AND deleted_at IS NULL
);

UPDATE groups
SET
    description = 'Trial subscription group: 100 USD daily quota for 1 day.',
    platform = 'openai',
    subscription_type = 'subscription',
    daily_limit_usd = 100.00000000,
    weekly_limit_usd = NULL,
    monthly_limit_usd = NULL,
    default_validity_days = 1,
    status = 'active',
    rate_multiplier = 1.0,
    sort_order = 100,
    updated_at = NOW()
WHERE name = 'JP 100USD Daily Trial' AND deleted_at IS NULL;

INSERT INTO groups (
    name,
    description,
    platform,
    subscription_type,
    daily_limit_usd,
    weekly_limit_usd,
    monthly_limit_usd,
    default_validity_days,
    status,
    rate_multiplier,
    sort_order,
    created_at,
    updated_at
)
SELECT
    'JP 50USD Daily Weekly',
    'Weekly subscription group: 50 USD daily quota for 7 days, 350 USD total weekly quota.',
    'openai',
    'subscription',
    50.00000000,
    350.00000000,
    NULL,
    7,
    'active',
    1.0,
    110,
    NOW(),
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM groups WHERE name = 'JP 50USD Daily Weekly' AND deleted_at IS NULL
);

UPDATE groups
SET
    description = 'Weekly subscription group: 50 USD daily quota for 7 days, 350 USD total weekly quota.',
    platform = 'openai',
    subscription_type = 'subscription',
    daily_limit_usd = 50.00000000,
    weekly_limit_usd = 350.00000000,
    monthly_limit_usd = NULL,
    default_validity_days = 7,
    status = 'active',
    rate_multiplier = 1.0,
    sort_order = 110,
    updated_at = NOW()
WHERE name = 'JP 50USD Daily Weekly' AND deleted_at IS NULL;

-- Visible subscription plans.
INSERT INTO subscription_plans (
    group_id,
    name,
    description,
    price,
    original_price,
    validity_days,
    validity_unit,
    features,
    product_name,
    for_sale,
    sort_order,
    created_at,
    updated_at
)
SELECT
    g.id,
    '100 USD Daily Trial Card',
    '1 day trial card with 100 USD daily quota.',
    100.00,
    NULL,
    1,
    'day',
    '100 USD daily quota;1 day validity',
    '100 USD Daily Trial Card',
    TRUE,
    100,
    NOW(),
    NOW()
FROM groups g
WHERE g.name = 'JP 100USD Daily Trial'
  AND g.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM subscription_plans WHERE name = '100 USD Daily Trial Card'
  );

UPDATE subscription_plans
SET
    group_id = (SELECT id FROM groups WHERE name = 'JP 100USD Daily Trial' AND deleted_at IS NULL LIMIT 1),
    description = '1 day trial card with 100 USD daily quota.',
    price = 100.00,
    original_price = NULL,
    validity_days = 1,
    validity_unit = 'day',
    features = '100 USD daily quota;1 day validity',
    product_name = '100 USD Daily Trial Card',
    for_sale = TRUE,
    sort_order = 100,
    updated_at = NOW()
WHERE name = '100 USD Daily Trial Card';

INSERT INTO subscription_plans (
    group_id,
    name,
    description,
    price,
    original_price,
    validity_days,
    validity_unit,
    features,
    product_name,
    for_sale,
    sort_order,
    created_at,
    updated_at
)
SELECT
    g.id,
    '50 USD Daily Weekly Card',
    '7 day weekly card with 50 USD daily quota, 350 USD total weekly quota.',
    350.00,
    NULL,
    7,
    'day',
    '50 USD daily quota;7 day validity;350 USD total weekly quota',
    '50 USD Daily Weekly Card',
    TRUE,
    110,
    NOW(),
    NOW()
FROM groups g
WHERE g.name = 'JP 50USD Daily Weekly'
  AND g.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM subscription_plans WHERE name = '50 USD Daily Weekly Card'
  );

UPDATE subscription_plans
SET
    group_id = (SELECT id FROM groups WHERE name = 'JP 50USD Daily Weekly' AND deleted_at IS NULL LIMIT 1),
    description = '7 day weekly card with 50 USD daily quota, 350 USD total weekly quota.',
    price = 350.00,
    original_price = NULL,
    validity_days = 7,
    validity_unit = 'day',
    features = '50 USD daily quota;7 day validity;350 USD total weekly quota',
    product_name = '50 USD Daily Weekly Card',
    for_sale = TRUE,
    sort_order = 110,
    updated_at = NOW()
WHERE name = '50 USD Daily Weekly Card';

-- Redeem codes. Keep total seeded count idempotent by notes marker.
WITH target_group AS (
    SELECT id FROM groups WHERE name = 'JP 100USD Daily Trial' AND deleted_at IS NULL LIMIT 1
),
needed AS (
    SELECT GREATEST(
        0,
        20 - (
            SELECT COUNT(*)
            FROM redeem_codes
            WHERE type = 'subscription'
              AND group_id = (SELECT id FROM target_group)
              AND notes = 'JP seed: 100 USD daily trial card'
        )
    )::INT AS count
),
generated AS (
    SELECT
        'JPTR' || UPPER(SUBSTRING(MD5(RANDOM()::TEXT || CLOCK_TIMESTAMP()::TEXT || gs::TEXT), 1, 20)) AS code,
        (SELECT id FROM target_group) AS group_id
    FROM GENERATE_SERIES(1, (SELECT count FROM needed)) AS gs
)
INSERT INTO redeem_codes (
    code,
    type,
    value,
    status,
    group_id,
    validity_days,
    notes,
    created_at
)
SELECT
    code,
    'subscription',
    100.00000000,
    'unused',
    group_id,
    1,
    'JP seed: 100 USD daily trial card',
    NOW()
FROM generated
WHERE group_id IS NOT NULL
ON CONFLICT (code) DO NOTHING;

WITH target_group AS (
    SELECT id FROM groups WHERE name = 'JP 50USD Daily Weekly' AND deleted_at IS NULL LIMIT 1
),
needed AS (
    SELECT GREATEST(
        0,
        5 - (
            SELECT COUNT(*)
            FROM redeem_codes
            WHERE type = 'subscription'
              AND group_id = (SELECT id FROM target_group)
              AND notes = 'JP seed: 50 USD daily weekly card'
        )
    )::INT AS count
),
generated AS (
    SELECT
        'JPWK' || UPPER(SUBSTRING(MD5(RANDOM()::TEXT || CLOCK_TIMESTAMP()::TEXT || gs::TEXT), 1, 20)) AS code,
        (SELECT id FROM target_group) AS group_id
    FROM GENERATE_SERIES(1, (SELECT count FROM needed)) AS gs
)
INSERT INTO redeem_codes (
    code,
    type,
    value,
    status,
    group_id,
    validity_days,
    notes,
    created_at
)
SELECT
    code,
    'subscription',
    350.00000000,
    'unused',
    group_id,
    7,
    'JP seed: 50 USD daily weekly card',
    NOW()
FROM generated
WHERE group_id IS NOT NULL
ON CONFLICT (code) DO NOTHING;
