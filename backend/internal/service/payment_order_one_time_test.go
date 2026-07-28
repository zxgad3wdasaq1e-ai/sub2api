package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestGetOneTimeSubscriptionStates(t *testing.T) {
	ctx := context.Background()
	client := newOneTimeSubscriptionTestClient(t)
	user, err := client.User.Create().
		SetEmail("one-time-subscription@example.com").
		SetPasswordHash("hash").
		SetUsername("one-time-subscription").
		Save(ctx)
	require.NoError(t, err)

	createOrder := func(groupID int64, status string, paidAt *time.Time) {
		t.Helper()
		builder := client.PaymentOrder.Create().
			SetUserID(user.ID).
			SetUserEmail(user.Email).
			SetUserName(user.Username).
			SetAmount(1).
			SetPayAmount(1).
			SetFeeRate(0).
			SetRechargeCode(fmt.Sprintf("ONE-TIME-%d", groupID)).
			SetOutTradeNo(fmt.Sprintf("sub2_one_time_%d", groupID)).
			SetPaymentType(payment.TypeAlipay).
			SetPaymentTradeNo("").
			SetOrderType(payment.OrderTypeSubscription).
			SetSubscriptionGroupID(groupID).
			SetSubscriptionDays(30).
			SetStatus(status).
			SetExpiresAt(time.Now().Add(time.Hour)).
			SetClientIP("127.0.0.1").
			SetSrcHost("api.example.com")
		if paidAt != nil {
			builder.SetPaidAt(*paidAt)
		}
		_, err := builder.Save(ctx)
		require.NoError(t, err)
	}

	now := time.Now()
	createOrder(10, OrderStatusPending, nil)
	createOrder(11, OrderStatusCancelled, nil)
	createOrder(12, OrderStatusCompleted, nil)
	createOrder(13, OrderStatusFailed, &now)
	createOrder(14, "FUTURE_PAID_STATE", nil)
	redeemGroup, err := client.Group.Create().SetName("Redeemed one-time group").Save(ctx)
	require.NoError(t, err)
	_, err = client.RedeemCode.Create().
		SetCode("ONE-TIME-REDEEMED").
		SetType(RedeemTypeSubscription).
		SetStatus(StatusUsed).
		SetUsedBy(user.ID).
		SetUsedAt(now).
		SetGroupID(redeemGroup.ID).
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}
	states, err := svc.GetOneTimeSubscriptionStates(ctx, user.ID, []int64{10, 11, 12, 13, 14, redeemGroup.ID})
	require.NoError(t, err)
	require.Equal(t, OneTimeSubscriptionState{Pending: true}, states[10])
	require.Equal(t, OneTimeSubscriptionState{}, states[11])
	require.Equal(t, OneTimeSubscriptionState{Purchased: true}, states[12])
	require.Equal(t, OneTimeSubscriptionState{Purchased: true}, states[13])
	require.Equal(t, OneTimeSubscriptionState{Purchased: true}, states[14])
	require.Equal(t, OneTimeSubscriptionState{Purchased: true}, states[redeemGroup.ID])
}

func TestOneTimeSubscriptionRedeemUniqueConstraint(t *testing.T) {
	ctx := context.Background()
	client := newOneTimeSubscriptionTestClient(t)
	user, err := client.User.Create().
		SetEmail("one-time-redeem-constraint@example.com").
		SetPasswordHash("hash").
		SetUsername("one-time-redeem-constraint").
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().SetName("One-time redeem constraint").Save(ctx)
	require.NoError(t, err)

	createRedeem := func(code string) error {
		_, err := client.RedeemCode.Create().
			SetCode(code).
			SetType(RedeemTypeSubscription).
			SetStatus(StatusUsed).
			SetUsedBy(user.ID).
			SetUsedAt(time.Now()).
			SetGroupID(group.ID).
			SetOneTimeSubscription(true).
			Save(ctx)
		return err
	}

	require.NoError(t, createRedeem("ONE-TIME-REDEEM-1"))
	require.True(t, dbent.IsConstraintError(createRedeem("ONE-TIME-REDEEM-2")))
}

func newOneTimeSubscriptionTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:payment_order_one_time?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestValidateOneTimeSubscriptionState(t *testing.T) {
	require.NoError(t, validateOneTimeSubscriptionState(OneTimeSubscriptionState{}))
	require.Equal(t, "ONE_TIME_SUBSCRIPTION_PENDING", infraerrors.Reason(validateOneTimeSubscriptionState(OneTimeSubscriptionState{Pending: true})))
	require.Equal(t, "ONE_TIME_SUBSCRIPTION_ALREADY_PURCHASED", infraerrors.Reason(validateOneTimeSubscriptionState(OneTimeSubscriptionState{Purchased: true})))
}

func TestOneTimeSubscriptionOrderUniqueConstraint(t *testing.T) {
	ctx := context.Background()
	client := newOneTimeSubscriptionTestClient(t)
	user, err := client.User.Create().
		SetEmail("one-time-constraint@example.com").
		SetPasswordHash("hash").
		SetUsername("one-time-constraint").
		Save(ctx)
	require.NoError(t, err)

	orderSequence := 0
	createOrder := func() (*dbent.PaymentOrder, error) {
		orderSequence++
		return client.PaymentOrder.Create().
			SetUserID(user.ID).
			SetUserEmail(user.Email).
			SetUserName(user.Username).
			SetAmount(1).
			SetPayAmount(1).
			SetFeeRate(0).
			SetRechargeCode(fmt.Sprintf("ONE-TIME-CONSTRAINT-%d", orderSequence)).
			SetOutTradeNo(fmt.Sprintf("sub2_one_time_constraint_%d", orderSequence)).
			SetPaymentType(payment.TypeAlipay).
			SetPaymentTradeNo("").
			SetOrderType(payment.OrderTypeSubscription).
			SetSubscriptionGroupID(20).
			SetSubscriptionDays(30).
			SetOneTimeSubscription(true).
			SetStatus(OrderStatusPending).
			SetExpiresAt(time.Now().Add(time.Hour)).
			SetClientIP("127.0.0.1").
			SetSrcHost("api.example.com").
			Save(ctx)
	}

	first, err := createOrder()
	require.NoError(t, err)
	_, err = createOrder()
	require.True(t, dbent.IsConstraintError(err))

	err = client.PaymentOrder.UpdateOneID(first.ID).SetStatus(OrderStatusCancelled).Exec(ctx)
	require.NoError(t, err)
	second, err := createOrder()
	require.NoError(t, err)

	err = client.PaymentOrder.UpdateOneID(second.ID).
		SetStatus(OrderStatusFailed).
		SetPaidAt(time.Now()).
		Exec(ctx)
	require.NoError(t, err)
	_, err = createOrder()
	require.True(t, dbent.IsConstraintError(err))
}
