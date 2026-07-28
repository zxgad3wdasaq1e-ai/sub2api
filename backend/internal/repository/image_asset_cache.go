package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const imageAssetCacheTTL = 5 * time.Minute

type imageAssetCache struct {
	rdb *redis.Client
}

func NewImageAssetCache(rdb *redis.Client) service.ImageAssetCache {
	return &imageAssetCache{rdb: rdb}
}

func imageAssetCacheKey(userID int64) string {
	return "image_assets:user:" + strconv.FormatInt(userID, 10)
}

func (c *imageAssetCache) Get(ctx context.Context, userID int64) ([]service.ImageAsset, bool, error) {
	if c == nil || c.rdb == nil || userID <= 0 {
		return nil, false, nil
	}
	payload, err := c.rdb.Get(ctx, imageAssetCacheKey(userID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var assets []service.ImageAsset
	if err := json.Unmarshal(payload, &assets); err != nil {
		_ = c.rdb.Del(ctx, imageAssetCacheKey(userID)).Err()
		return nil, false, err
	}
	return assets, true, nil
}

func (c *imageAssetCache) Set(ctx context.Context, userID int64, assets []service.ImageAsset) error {
	if c == nil || c.rdb == nil || userID <= 0 {
		return nil
	}
	payload, err := json.Marshal(assets)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, imageAssetCacheKey(userID), payload, imageAssetCacheTTL).Err()
}

func (c *imageAssetCache) Invalidate(ctx context.Context, userID int64) error {
	if c == nil || c.rdb == nil || userID <= 0 {
		return nil
	}
	return c.rdb.Del(ctx, imageAssetCacheKey(userID)).Err()
}
