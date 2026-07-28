ALTER TABLE redeem_codes
    ADD COLUMN IF NOT EXISTS one_time_subscription BOOLEAN NOT NULL DEFAULT FALSE;

WITH first_one_time_redemptions AS (
    SELECT MIN(rc.id) AS redeem_code_id
    FROM redeem_codes rc
    INNER JOIN groups g ON g.id = rc.group_id
    WHERE rc.type = 'subscription'
      AND rc.status = 'used'
      AND rc.used_by IS NOT NULL
      AND rc.group_id IS NOT NULL
      AND g.one_time_subscription = TRUE
    GROUP BY rc.used_by, rc.group_id
)
UPDATE redeem_codes rc
SET one_time_subscription = TRUE
FROM first_one_time_redemptions first_use
WHERE rc.id = first_use.redeem_code_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_redeem_codes_one_time_subscription_used
    ON redeem_codes (used_by, group_id)
    WHERE one_time_subscription = TRUE
      AND type = 'subscription'
      AND status = 'used'
      AND used_by IS NOT NULL
      AND group_id IS NOT NULL;

COMMENT ON COLUMN redeem_codes.one_time_subscription IS
    'Snapshot of the subscription group one-time purchase rule when redeemed';
