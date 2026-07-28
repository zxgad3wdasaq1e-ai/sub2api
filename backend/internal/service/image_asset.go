package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	defaultImageAssetRetention       = 7 * 24 * time.Hour
	defaultImageAssetListLimit       = 100
	defaultImageAssetCleanupInterval = 30 * time.Minute
	defaultImageAssetCleanupBatch    = 100
)

var (
	ErrImageAssetNotFound          = infraerrors.New(http.StatusNotFound, "IMAGE_ASSET_NOT_FOUND", "image asset not found")
	ErrImageAssetUnavailable       = infraerrors.New(http.StatusServiceUnavailable, "IMAGE_ASSET_UNAVAILABLE", "image asset storage is unavailable")
	ErrImageAssetObjectUnavailable = infraerrors.New(http.StatusServiceUnavailable, "IMAGE_ASSET_OBJECT_UNAVAILABLE", "image object storage is unavailable")
)

// ImageTaskMetadata is persisted with a generation job so the user library can
// be reconstructed independently of browser storage.
type ImageTaskMetadata struct {
	Prompt       string
	Model        string
	Mode         string
	Size         string
	Quality      string
	OutputFormat string
}

type ImageGenerationJob struct {
	TaskID       string
	UserID       int64
	APIKeyID     int64
	Prompt       string
	Model        string
	Mode         string
	Size         string
	Quality      string
	OutputFormat string
	Status       string
	ErrorMessage string
	CreatedAt    time.Time
	CompletedAt  *time.Time
	ExpiresAt    time.Time
}

// ImageAsset is the durable image-library record. ObjectKey and UserID are
// internal fields; handlers expose only authenticated Sub2API proxy URLs.
type ImageAsset struct {
	ID           string    `json:"id"`
	TaskID       string    `json:"task_id"`
	UserID       int64     `json:"-"`
	ImageIndex   int       `json:"image_index"`
	ObjectKey    string    `json:"-"`
	ContentType  string    `json:"content_type"`
	ByteSize     int64     `json:"byte_size"`
	Prompt       string    `json:"prompt"`
	Model        string    `json:"model"`
	Mode         string    `json:"mode"`
	Size         string    `json:"size"`
	Quality      string    `json:"quality"`
	OutputFormat string    `json:"output_format"`
	URL          string    `json:"url"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type CreateImageGenerationJobParams struct {
	TaskID    string
	UserID    int64
	APIKeyID  int64
	Metadata  ImageTaskMetadata
	Status    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type CreateImageAssetParams struct {
	ID          string
	ImageIndex  int
	ObjectKey   string
	ContentType string
	ByteSize    int64
}

type ImageAssetRepository interface {
	CreateImageGenerationJob(ctx context.Context, params CreateImageGenerationJobParams) error
	DeleteImageGenerationJob(ctx context.Context, taskID string) error
	CompleteImageGenerationJob(ctx context.Context, taskID string, completedAt time.Time, assets []CreateImageAssetParams) (int64, error)
	FailImageGenerationJob(ctx context.Context, taskID, errorMessage string, completedAt time.Time) (int64, error)
	ListImageAssetsForUser(ctx context.Context, userID int64, now time.Time, limit int) ([]ImageAsset, error)
	GetImageAssetForUser(ctx context.Context, userID int64, assetID string) (*ImageAsset, error)
	GetImageAssetForAPIKey(ctx context.Context, userID, apiKeyID int64, assetID string) (*ImageAsset, error)
	ListImageAssetsForTaskOwner(ctx context.Context, userID, apiKeyID int64, taskID string, now time.Time) ([]ImageAsset, error)
	DeleteImageAssetForUser(ctx context.Context, userID int64, assetID string) error
	ListExpiredImageAssets(ctx context.Context, now time.Time, limit int) ([]ImageAsset, error)
	DeleteImageAssetByID(ctx context.Context, assetID string) error
	DeleteExpiredImageGenerationJobs(ctx context.Context, now time.Time) error
}

type ImageAssetCache interface {
	Get(ctx context.Context, userID int64) ([]ImageAsset, bool, error)
	Set(ctx context.Context, userID int64, assets []ImageAsset) error
	Invalidate(ctx context.Context, userID int64) error
}

type ImageAssetService struct {
	repo      ImageAssetRepository
	cache     ImageAssetCache
	resolve   ImageStorageResolver
	retention time.Duration
}

func NewImageAssetService(repo ImageAssetRepository, cache ImageAssetCache, settings *ImageStorageSettingService) *ImageAssetService {
	var resolve ImageStorageResolver
	if settings != nil {
		resolve = settings.Resolver()
	}
	return &ImageAssetService{repo: repo, cache: cache, resolve: resolve, retention: defaultImageAssetRetention}
}

func (s *ImageAssetService) CreateJob(ctx context.Context, taskID string, owner ImageTaskOwner, metadata ImageTaskMetadata, createdAt time.Time) error {
	if s == nil || s.repo == nil {
		return ErrImageAssetUnavailable
	}
	metadata.Mode = strings.TrimSpace(metadata.Mode)
	if metadata.Mode == "" {
		metadata.Mode = "text"
	}
	return s.repo.CreateImageGenerationJob(ctx, CreateImageGenerationJobParams{
		TaskID: taskID, UserID: owner.UserID, APIKeyID: owner.APIKeyID,
		Metadata: metadata, Status: ImageTaskStatusProcessing,
		CreatedAt: createdAt, ExpiresAt: createdAt.Add(s.retention),
	})
}

func (s *ImageAssetService) DeleteJob(ctx context.Context, taskID string) error {
	if s == nil || s.repo == nil {
		return ErrImageAssetUnavailable
	}
	return s.repo.DeleteImageGenerationJob(ctx, taskID)
}

func (s *ImageAssetService) CompleteJob(ctx context.Context, taskID string, objects []StoredImageObject) error {
	if s == nil || s.repo == nil {
		return ErrImageAssetUnavailable
	}
	assets := make([]CreateImageAssetParams, 0, len(objects))
	for _, object := range objects {
		assets = append(assets, CreateImageAssetParams{
			ID:         "imgasset_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
			ImageIndex: object.Index, ObjectKey: object.ObjectKey,
			ContentType: object.ContentType, ByteSize: object.ByteSize,
		})
	}
	userID, err := s.repo.CompleteImageGenerationJob(ctx, taskID, time.Now().UTC(), assets)
	if err != nil {
		return err
	}
	s.invalidate(ctx, userID)
	return nil
}

func (s *ImageAssetService) FailJob(ctx context.Context, taskID, message string) error {
	if s == nil || s.repo == nil {
		return ErrImageAssetUnavailable
	}
	userID, err := s.repo.FailImageGenerationJob(ctx, taskID, strings.TrimSpace(message), time.Now().UTC())
	if err != nil {
		return err
	}
	s.invalidate(ctx, userID)
	return nil
}

func (s *ImageAssetService) List(ctx context.Context, userID int64) ([]ImageAsset, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrImageAssetUnavailable
	}
	if s.cache != nil {
		if cached, ok, err := s.cache.Get(ctx, userID); err == nil && ok {
			return stripImageAssetURLs(cached), nil
		}
	}
	assets, err := s.repo.ListImageAssetsForUser(ctx, userID, time.Now().UTC(), defaultImageAssetListLimit)
	if err != nil {
		return nil, ErrImageAssetUnavailable.WithCause(err)
	}
	assets = stripImageAssetURLs(assets)
	if s.cache != nil {
		if err := s.cache.Set(ctx, userID, assets); err != nil {
			logger.L().Warn("image_asset.cache_set_failed", zap.Int64("user_id", userID), zap.Error(err))
		}
	}
	return assets, nil
}

// ListForTaskOwner returns generated assets for one task after the caller's
// user and API key ownership has been verified by the gateway.
func (s *ImageAssetService) ListForTaskOwner(ctx context.Context, userID, apiKeyID int64, taskID string) ([]ImageAsset, error) {
	if s == nil || s.repo == nil || userID <= 0 || apiKeyID <= 0 || strings.TrimSpace(taskID) == "" {
		return nil, ErrImageAssetUnavailable
	}
	assets, err := s.repo.ListImageAssetsForTaskOwner(ctx, userID, apiKeyID, strings.TrimSpace(taskID), time.Now().UTC())
	if err != nil {
		return nil, ErrImageAssetUnavailable.WithCause(err)
	}
	return stripImageAssetURLs(assets), nil
}

// Open verifies the current user's ownership and streams the asset through
// the configured object storage. Returned bodies must be closed by callers.
func (s *ImageAssetService) Open(ctx context.Context, userID int64, assetID string) (*ImageAsset, io.ReadCloser, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, nil, ErrImageAssetUnavailable
	}
	asset, err := s.repo.GetImageAssetForUser(ctx, userID, strings.TrimSpace(assetID))
	if err != nil {
		return nil, nil, imageAssetLookupError(err)
	}
	return s.openAssetObject(ctx, asset)
}

// OpenForAPIKey verifies both user and API key ownership before streaming an
// asset for a public Images API caller.
func (s *ImageAssetService) OpenForAPIKey(ctx context.Context, userID, apiKeyID int64, assetID string) (*ImageAsset, io.ReadCloser, error) {
	if s == nil || s.repo == nil || userID <= 0 || apiKeyID <= 0 {
		return nil, nil, ErrImageAssetUnavailable
	}
	asset, err := s.repo.GetImageAssetForAPIKey(ctx, userID, apiKeyID, strings.TrimSpace(assetID))
	if err != nil {
		return nil, nil, imageAssetLookupError(err)
	}
	return s.openAssetObject(ctx, asset)
}

func (s *ImageAssetService) openAssetObject(ctx context.Context, asset *ImageAsset) (*ImageAsset, io.ReadCloser, error) {
	if asset == nil || !asset.ExpiresAt.After(time.Now().UTC()) {
		return nil, nil, ErrImageAssetNotFound
	}
	uploader := s.currentUploader()
	if uploader == nil {
		return nil, nil, ErrImageAssetObjectUnavailable
	}
	body, contentType, contentLength, err := uploader.Open(ctx, asset.ObjectKey)
	if err != nil {
		return nil, nil, ErrImageAssetObjectUnavailable.WithCause(err)
	}
	copy := *asset
	if strings.TrimSpace(contentType) != "" {
		copy.ContentType = contentType
	}
	if contentLength >= 0 {
		copy.ByteSize = contentLength
	}
	return &copy, body, nil
}

func imageAssetLookupError(err error) error {
	if errors.Is(err, ErrImageAssetNotFound) {
		return ErrImageAssetNotFound
	}
	return ErrImageAssetUnavailable.WithCause(err)
}

func stripImageAssetURLs(assets []ImageAsset) []ImageAsset {
	for i := range assets {
		assets[i].URL = ""
	}
	return assets
}

func (s *ImageAssetService) Delete(ctx context.Context, userID int64, assetID string) error {
	if s == nil || s.repo == nil || userID <= 0 {
		return ErrImageAssetUnavailable
	}
	asset, err := s.repo.GetImageAssetForUser(ctx, userID, strings.TrimSpace(assetID))
	if err != nil {
		if errors.Is(err, ErrImageAssetNotFound) {
			return ErrImageAssetNotFound
		}
		return ErrImageAssetUnavailable.WithCause(err)
	}
	uploader := s.currentUploader()
	if uploader == nil {
		return ErrImageAssetObjectUnavailable
	}
	if err := uploader.Delete(ctx, asset.ObjectKey); err != nil {
		return ErrImageAssetObjectUnavailable.WithCause(err)
	}
	if err := s.repo.DeleteImageAssetForUser(ctx, userID, asset.ID); err != nil {
		if errors.Is(err, ErrImageAssetNotFound) {
			return ErrImageAssetNotFound
		}
		return ErrImageAssetUnavailable.WithCause(err)
	}
	s.invalidate(ctx, userID)
	return nil
}

func (s *ImageAssetService) currentUploader() *ImageResultUploader {
	if s == nil || s.resolve == nil {
		return nil
	}
	uploader, _ := s.resolve()
	return uploader
}

func (s *ImageAssetService) invalidate(ctx context.Context, userID int64) {
	if s == nil || s.cache == nil || userID <= 0 {
		return
	}
	if err := s.cache.Invalidate(ctx, userID); err != nil {
		logger.L().Warn("image_asset.cache_invalidate_failed", zap.Int64("user_id", userID), zap.Error(err))
	}
}

type ImageAssetCleanupService struct {
	assets   *ImageAssetService
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
	mu       sync.Mutex
}

func NewImageAssetCleanupService(assets *ImageAssetService) *ImageAssetCleanupService {
	return &ImageAssetCleanupService{assets: assets, interval: defaultImageAssetCleanupInterval}
}

func (s *ImageAssetCleanupService) RunOnce(ctx context.Context, now time.Time) error {
	if s == nil || s.assets == nil || s.assets.repo == nil {
		return nil
	}
	uploader := s.assets.currentUploader()
	if uploader == nil {
		// Storage may intentionally be disabled before any image has ever been
		// generated. Keep the periodic worker quiet until a configured storage
		// binding is available again.
		return nil
	}
	assets, err := s.assets.repo.ListExpiredImageAssets(ctx, now, defaultImageAssetCleanupBatch)
	if err != nil {
		return err
	}
	for _, asset := range assets {
		if err := uploader.Delete(ctx, asset.ObjectKey); err != nil {
			logger.L().Warn("image_asset.cleanup_object_failed", zap.String("asset_id", asset.ID), zap.Error(err))
			continue
		}
		if err := s.assets.repo.DeleteImageAssetByID(ctx, asset.ID); err != nil {
			logger.L().Warn("image_asset.cleanup_record_failed", zap.String("asset_id", asset.ID), zap.Error(err))
			continue
		}
		s.assets.invalidate(ctx, asset.UserID)
	}
	return s.assets.repo.DeleteExpiredImageGenerationJobs(ctx, now)
}

func (s *ImageAssetCleanupService) Start() {
	if s == nil || s.assets == nil || s.interval <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			if err := s.RunOnce(ctx, time.Now().UTC()); err != nil && !errors.Is(err, context.Canceled) {
				logger.L().Warn("image_asset.cleanup_failed", zap.Error(err))
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *ImageAssetCleanupService) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel = nil
	s.done = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}
