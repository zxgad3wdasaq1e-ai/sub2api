//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedeemSnapshotsOneTimeSubscriptionRule(t *testing.T) {
	ctx := context.Background()
	client := newOneTimeSubscriptionTestClient(t)
	groupID := int64(88)
	useErr := errors.New("stop after use")
	redeemRepo := &redeemRejectRepo{
		code: RedeemCode{
			ID:           1,
			Code:         "ONE-TIME-SNAPSHOT",
			Type:         RedeemTypeSubscription,
			Status:       StatusUnused,
			GroupID:      &groupID,
			ValidityDays: 30,
		},
		useErr: useErr,
	}
	subscriptionService := &SubscriptionService{groupRepo: &subscriptionGroupRepoStub{group: &Group{
		ID:                  groupID,
		SubscriptionType:    SubscriptionTypeSubscription,
		OneTimeSubscription: true,
	}}}
	svc := &RedeemService{
		redeemRepo:          redeemRepo,
		userRepo:            &userRepoStub{user: &User{ID: 7}},
		subscriptionService: subscriptionService,
		entClient:           client,
	}

	got, err := svc.Redeem(ctx, 7, redeemRepo.code.Code)
	require.Nil(t, got)
	require.ErrorIs(t, err, useErr)
	require.True(t, redeemRepo.useCalled)
	require.True(t, redeemRepo.usedOneTimeSnapshot)
	require.Equal(t, StatusUnused, redeemRepo.code.Status)
}
