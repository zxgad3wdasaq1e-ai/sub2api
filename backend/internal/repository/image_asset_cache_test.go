package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestImageAssetCacheRoundTripAndInvalidation(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := NewImageAssetCache(rdb)
	assets := []service.ImageAsset{{
		ID:        "imgasset_1",
		TaskID:    "imgtask_1",
		Prompt:    "city at night",
		URL:       "https://cdn.example.test/images/imgtask_1-0.png",
		CreatedAt: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC),
	}}

	require.NoError(t, cache.Set(context.Background(), 42, assets))
	got, ok, err := cache.Get(context.Background(), 42)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, assets, got)
	require.Equal(t, imageAssetCacheTTL, mr.TTL(imageAssetCacheKey(42)))

	require.NoError(t, cache.Invalidate(context.Background(), 42))
	_, ok, err = cache.Get(context.Background(), 42)
	require.NoError(t, err)
	require.False(t, ok)
}
