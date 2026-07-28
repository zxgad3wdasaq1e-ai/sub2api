ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS one_time_subscription BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS one_time_subscription BOOLEAN NOT NULL DEFAULT FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_one_time_subscription_active
    ON payment_orders (user_id, subscription_group_id)
    WHERE one_time_subscription = TRUE
      AND subscription_group_id IS NOT NULL
      AND (status NOT IN ('CANCELLED', 'EXPIRED', 'FAILED') OR paid_at IS NOT NULL);

COMMENT ON COLUMN groups.one_time_subscription IS
    'When true, each user may purchase this subscription group only once';
COMMENT ON COLUMN payment_orders.one_time_subscription IS
    'Snapshot of the group one-time purchase rule when this order was created';
