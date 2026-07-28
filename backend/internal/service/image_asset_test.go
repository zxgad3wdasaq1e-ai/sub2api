package service

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type imageAssetTestRepo struct {
	assets      []ImageAsset
	getAsset    *ImageAsset
	listCalls   int
	deletedUser int64
	deletedID   string
}

func (r *imageAssetTestRepo) CreateImageGenerationJob(context.Context, CreateImageGenerationJobParams) error {
	return nil
}
func (r *imageAssetTestRepo) DeleteImageGenerationJob(context.Context, string) error { return nil }
func (r *imageAssetTestRepo) CompleteImageGenerationJob(_ context.Context, _ string, _ time.Time, _ []CreateImageAssetParams) (int64, error) {
	return 7, nil
}
func (r *imageAssetTestRepo) FailImageGenerationJob(_ context.Context, _ string, _ string, _ time.Time) (int64, error) {
	return 7, nil
}
func (r *imageAssetTestRepo) ListImageAssetsForUser(_ context.Context, _ int64, _ time.Time, _ int) ([]ImageAsset, error) {
	r.listCalls++
	return append([]ImageAsset(nil), r.assets...), nil
}
func (r *imageAssetTestRepo) GetImageAssetForUser(_ context.Context, _ int64, _ string) (*ImageAsset, error) {
	if r.getAsset == nil {
		return nil, ErrImageAssetNotFound
	}
	copy := *r.getAsset
	return &copy, nil
}
func (r *imageAssetTestRepo) GetImageAssetForAPIKey(_ context.Context, _ int64, _ int64, _ string) (*ImageAsset, error) {
	if r.getAsset == nil {
		return nil, ErrImageAssetNotFound
	}
	copy := *r.getAsset
	return &copy, nil
}
func (r *imageAssetTestRepo) ListImageAssetsForTaskOwner(_ context.Context, _ int64, _ int64, _ string, _ time.Time) ([]ImageAsset, error) {
	return append([]ImageAsset(nil), r.assets...), nil
}
func (r *imageAssetTestRepo) DeleteImageAssetForUser(_ context.Context, userID int64, assetID string) error {
	r.deletedUser, r.deletedID = userID, assetID
	return nil
}
func (r *imageAssetTestRepo) ListExpiredImageAssets(context.Context, time.Time, int) ([]ImageAsset, error) {
	return nil, nil
}
func (r *imageAssetTestRepo) DeleteImageAssetByID(context.Context, string) error { return nil }
func (r *imageAssetTestRepo) DeleteExpiredImageGenerationJobs(context.Context, time.Time) error {
	return nil
}

type imageAssetTestCache struct {
	assets      []ImageAsset
	hit         bool
	sets        int
	invalidated int64
}

func (c *imageAssetTestCache) Get(context.Context, int64) ([]ImageAsset, bool, error) {
	return append([]ImageAsset(nil), c.assets...), c.hit, nil
}
func (c *imageAssetTestCache) Set(_ context.Context, _ int64, assets []ImageAsset) error {
	c.sets++
	c.assets = append([]ImageAsset(nil), assets...)
	c.hit = true
	return nil
}
func (c *imageAssetTestCache) Invalidate(_ context.Context, userID int64) error {
	c.invalidated = userID
	c.hit = false
	return nil
}

type imageAssetTestStorage struct {
	deleted []string
	body    string
}

func (*imageAssetTestStorage) Save(context.Context, string, string, []byte) (string, error) {
	return "", nil
}
func (*imageAssetTestStorage) ResolveURL(_ context.Context, key string) (string, error) {
	return "https://cdn.example.test/" + key, nil
}
func (s *imageAssetTestStorage) Open(context.Context, string) (io.ReadCloser, string, int64, error) {
	return io.NopCloser(strings.NewReader(s.body)), "image/png", int64(len(s.body)), nil
}
func (s *imageAssetTestStorage) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

func newImageAssetTestService(repo ImageAssetRepository, cache ImageAssetCache, storage ImageStorage) *ImageAssetService {
	uploader := NewImageResultUploader(storage, "images/", 0, nil)
	return &ImageAssetService{
		repo: repo, cache: cache, retention: defaultImageAssetRetention,
		resolve: func() (*ImageResultUploader, bool) { return uploader, true },
	}
}

func TestImageAssetServiceListCachesAssetMetadataWithoutStorageURLs(t *testing.T) {
	repo := &imageAssetTestRepo{assets: []ImageAsset{{
		ID: "imgasset_1", UserID: 7, ObjectKey: "images/imgtask_1-0.png",
		CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour),
	}}}
	cache := &imageAssetTestCache{}
	svc := newImageAssetTestService(repo, cache, &imageAssetTestStorage{})

	first, err := svc.List(context.Background(), 7)
	require.NoError(t, err)
	require.Empty(t, first[0].URL)
	require.Equal(t, 1, repo.listCalls)
	require.Equal(t, 1, cache.sets)

	second, err := svc.List(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, 1, repo.listCalls, "second list should come from Redis cache")
}

func TestImageAssetServiceOpenStreamsOwnedObject(t *testing.T) {
	storage := &imageAssetTestStorage{body: "image-bytes"}
	repo := &imageAssetTestRepo{getAsset: &ImageAsset{
		ID: "imgasset_1", UserID: 7, ObjectKey: "images/imgtask_1-0.png",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}}
	svc := newImageAssetTestService(repo, nil, storage)

	asset, body, err := svc.Open(context.Background(), 7, "imgasset_1")
	require.NoError(t, err)
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	require.Equal(t, "image-bytes", string(data))
	require.Equal(t, "image/png", asset.ContentType)
	require.EqualValues(t, len(data), asset.ByteSize)
}

func TestImageAssetServiceDeleteRemovesObjectAndInvalidatesOwnerCache(t *testing.T) {
	storage := &imageAssetTestStorage{}
	repo := &imageAssetTestRepo{getAsset: &ImageAsset{ID: "imgasset_1", UserID: 7, ObjectKey: "images/imgtask_1-0.png"}}
	cache := &imageAssetTestCache{hit: true}
	svc := newImageAssetTestService(repo, cache, storage)

	require.NoError(t, svc.Delete(context.Background(), 7, "imgasset_1"))
	require.Equal(t, []string{"images/imgtask_1-0.png"}, storage.deleted)
	require.Equal(t, int64(7), repo.deletedUser)
	require.Equal(t, "imgasset_1", repo.deletedID)
	require.Equal(t, int64(7), cache.invalidated)
}
