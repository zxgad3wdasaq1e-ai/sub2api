package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration191AddsOneTimeSubscriptionRedeemGuards(t *testing.T) {
	content, err := FS.ReadFile("191_one_time_subscription_redeem_codes.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ALTER TABLE redeem_codes")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS one_time_subscription BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "MIN(rc.id) AS redeem_code_id")
	require.Contains(t, sql, "ON redeem_codes (used_by, group_id)")
	require.Contains(t, sql, "one_time_subscription = TRUE")
	require.Contains(t, sql, "type = 'subscription'")
	require.Contains(t, sql, "status = 'used'")
}
