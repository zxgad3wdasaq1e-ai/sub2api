package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type redeemRejectRepo struct {
	code                RedeemCode
	useCalled           bool
	usedOneTimeSnapshot bool
	useErr              error
}

func (r *redeemRejectRepo) Create(ctx context.Context, code *RedeemCode) error {
	panic("unexpected Create call")
}

func (r *redeemRejectRepo) CreateBatch(ctx context.Context, codes []RedeemCode) error {
	panic("unexpected CreateBatch call")
}

func (r *redeemRejectRepo) GetByID(ctx context.Context, id int64) (*RedeemCode, error) {
	if r.code.ID != id {
		return nil, ErrRedeemCodeNotFound
	}
	clone := r.code
	return &clone, nil
}

func (r *redeemRejectRepo) GetByCode(ctx context.Context, code string) (*RedeemCode, error) {
	if r.code.Code != code {
		return nil, ErrRedeemCodeNotFound
	}
	clone := r.code
	return &clone, nil
}

func (r *redeemRejectRepo) Update(ctx context.Context, code *RedeemCode) error {
	panic("unexpected Update call")
}

func (r *redeemRejectRepo) BatchUpdate(ctx context.Context, ids []int64, fields RedeemCodeBatchUpdateFields) (int64, error) {
	panic("unexpected BatchUpdate call")
}

func (r *redeemRejectRepo) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (r *redeemRejectRepo) Use(ctx context.Context, id, userID int64, oneTimeSubscription bool) error {
	r.useCalled = true
	r.usedOneTimeSnapshot = oneTimeSubscription
	if r.useErr != nil {
		return r.useErr
	}
	r.code.Status = StatusUsed
	r.code.UsedBy = &userID
	return nil
}

func TestRedeemRejectsPreviouslyRedeemedOneTimeSubscription(t *testing.T) {
	ctx := context.Background()
	client := newOneTimeSubscriptionTestClient(t)
	user, err := client.User.Create().
		SetEmail("one-time-redeem@example.com").
		SetPasswordHash("hash").
		SetUsername("one-time-redeem").
		Save(ctx)
	require.NoError(t, err)
	groupEntity, err := client.Group.Create().SetName("One-time redeem").Save(ctx)
	require.NoError(t, err)
	_, err = client.RedeemCode.Create().
		SetCode("ONE-TIME-REDEEM-FIRST").
		SetType(RedeemTypeSubscription).
		SetStatus(StatusUsed).
		SetUsedBy(user.ID).
		SetUsedAt(time.Now()).
		SetGroupID(groupEntity.ID).
		SetOneTimeSubscription(true).
		Save(ctx)
	require.NoError(t, err)

	groupID := groupEntity.ID
	redeemRepo := &redeemRejectRepo{code: RedeemCode{
		ID:           2,
		Code:         "ONE-TIME-REDEEM-SECOND",
		Type:         RedeemTypeSubscription,
		Status:       StatusUnused,
		GroupID:      &groupID,
		ValidityDays: 30,
	}}
	subscriptionService := &SubscriptionService{groupRepo: &subscriptionGroupRepoStub{group: &Group{
		ID:                  groupID,
		SubscriptionType:    SubscriptionTypeSubscription,
		OneTimeSubscription: true,
	}}}
	svc := &RedeemService{
		redeemRepo:          redeemRepo,
		subscriptionService: subscriptionService,
		entClient:           client,
	}

	got, err := svc.Redeem(ctx, user.ID, redeemRepo.code.Code)
	require.Nil(t, got)
	require.Equal(t, "ONE_TIME_SUBSCRIPTION_ALREADY_PURCHASED", infraerrors.Reason(err))
	require.False(t, redeemRepo.useCalled)
	require.Equal(t, StatusUnused, redeemRepo.code.Status)
}

func (r *redeemRejectRepo) List(ctx context.Context, params pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (r *redeemRejectRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (r *redeemRejectRepo) ListByUser(ctx context.Context, userID int64, limit int) ([]RedeemCode, error) {
	panic("unexpected ListByUser call")
}

func (r *redeemRejectRepo) ListByUserPaginated(ctx context.Context, userID int64, params pagination.PaginationParams, codeType string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserPaginated call")
}

func (r *redeemRejectRepo) SumPositiveBalanceByUser(ctx context.Context, userID int64) (float64, error) {
	panic("unexpected SumPositiveBalanceByUser call")
}

func TestRedeemRejectsInvitationCodeBeforeTransaction(t *testing.T) {
	ctx := context.Background()
	redeemRepo := &redeemRejectRepo{
		code: RedeemCode{
			ID:     1,
			Code:   "INVITE-001",
			Type:   RedeemTypeInvitation,
			Status: StatusUnused,
		},
	}
	redeemService := NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil)

	got, err := redeemService.Redeem(ctx, 2, redeemRepo.code.Code)

	require.Nil(t, got)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Equal(t, "REDEEM_CODE_UNSUPPORTED_TYPE", infraerrors.Reason(err))
	require.Equal(t, "invitation codes can only be used during registration", infraerrors.Message(err))
	require.False(t, redeemRepo.useCalled)
	require.Equal(t, StatusUnused, redeemRepo.code.Status)
	require.Nil(t, redeemRepo.code.UsedBy)
}
