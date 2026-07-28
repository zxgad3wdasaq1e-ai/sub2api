package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration185AddsOneTimeSubscriptionGuards(t *testing.T) {
	content, err := FS.ReadFile("185_one_time_subscriptions.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS one_time_subscription BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "ON payment_orders (user_id, subscription_group_id)")
	require.Contains(t, sql, "one_time_subscription = TRUE")
	require.Contains(t, sql, "status NOT IN ('CANCELLED', 'EXPIRED', 'FAILED') OR paid_at IS NOT NULL")
}
