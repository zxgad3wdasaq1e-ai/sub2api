package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type imageAssetSQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type imageAssetRepository struct {
	db  *sql.DB
	sql imageAssetSQLExecutor
}

func NewImageAssetRepository(db *sql.DB) service.ImageAssetRepository {
	return &imageAssetRepository{db: db, sql: db}
}

func (r *imageAssetRepository) CreateImageGenerationJob(ctx context.Context, params service.CreateImageGenerationJobParams) error {
	if r == nil || r.sql == nil {
		return errors.New("image asset database is unavailable")
	}
	if params.Status == "" {
		params.Status = service.ImageTaskStatusProcessing
	}
	if params.CreatedAt.IsZero() {
		params.CreatedAt = time.Now().UTC()
	}
	if params.ExpiresAt.IsZero() {
		params.ExpiresAt = params.CreatedAt.Add(7 * 24 * time.Hour)
	}
	_, err := r.sql.ExecContext(ctx, `
INSERT INTO image_generation_jobs
 (task_id, user_id, api_key_id, prompt, model, mode, size, quality, output_format, status, created_at, expires_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		params.TaskID, params.UserID, params.APIKeyID, params.Metadata.Prompt,
		params.Metadata.Model, params.Metadata.Mode, params.Metadata.Size,
		params.Metadata.Quality, params.Metadata.OutputFormat, params.Status,
		params.CreatedAt, params.ExpiresAt)
	return err
}

func (r *imageAssetRepository) DeleteImageGenerationJob(ctx context.Context, taskID string) error {
	if r == nil || r.sql == nil {
		return errors.New("image asset database is unavailable")
	}
	_, err := r.sql.ExecContext(ctx, `DELETE FROM image_generation_jobs WHERE task_id = $1`, taskID)
	return err
}

func (r *imageAssetRepository) CompleteImageGenerationJob(ctx context.Context, taskID string, completedAt time.Time, assets []service.CreateImageAssetParams) (int64, error) {
	if r == nil || r.sql == nil {
		return 0, errors.New("image asset database is unavailable")
	}
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	if r.db == nil {
		return r.completeImageGenerationJob(ctx, r.sql, taskID, completedAt, assets)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	userID, err := r.completeImageGenerationJob(ctx, tx, taskID, completedAt, assets)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return userID, nil
}

func (r *imageAssetRepository) completeImageGenerationJob(ctx context.Context, q imageAssetSQLExecutor, taskID string, completedAt time.Time, assets []service.CreateImageAssetParams) (int64, error) {
	var userID int64
	var expiresAt time.Time
	if err := q.QueryRowContext(ctx, `
UPDATE image_generation_jobs
SET status = 'completed', completed_at = $2, error_message = ''
WHERE task_id = $1
RETURNING user_id, expires_at`, taskID, completedAt).Scan(&userID, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, service.ErrImageAssetNotFound
		}
		return 0, err
	}
	for _, asset := range assets {
		if _, err := q.ExecContext(ctx, `
INSERT INTO image_assets
 (asset_id, task_id, user_id, image_index, object_key, content_type, byte_size, created_at, expires_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			asset.ID, taskID, userID, asset.ImageIndex, asset.ObjectKey,
			asset.ContentType, asset.ByteSize, completedAt, expiresAt); err != nil {
			return 0, err
		}
	}
	return userID, nil
}

func (r *imageAssetRepository) FailImageGenerationJob(ctx context.Context, taskID, errorMessage string, completedAt time.Time) (int64, error) {
	if r == nil || r.sql == nil {
		return 0, errors.New("image asset database is unavailable")
	}
	var userID int64
	err := r.sql.QueryRowContext(ctx, `
UPDATE image_generation_jobs
SET status = 'failed', completed_at = $2, error_message = $3
WHERE task_id = $1
RETURNING user_id`, taskID, completedAt, errorMessage).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrImageAssetNotFound
	}
	return userID, err
}

const imageAssetSelectSQL = `
SELECT a.asset_id, a.task_id, a.user_id, a.image_index, a.object_key,
       a.content_type, a.byte_size, a.created_at, a.expires_at,
       j.prompt, j.model, j.mode, j.size, j.quality, j.output_format
FROM image_assets a
JOIN image_generation_jobs j ON j.task_id = a.task_id
`

func (r *imageAssetRepository) ListImageAssetsForUser(ctx context.Context, userID int64, now time.Time, limit int) ([]service.ImageAsset, error) {
	if r == nil || r.sql == nil {
		return nil, errors.New("image asset database is unavailable")
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	rows, err := r.sql.QueryContext(ctx, imageAssetSelectSQL+`
WHERE a.user_id = $1 AND a.deleted_at IS NULL AND a.expires_at > $2
ORDER BY a.created_at DESC, a.image_index ASC LIMIT $3`, userID, now, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanImageAssets(rows)
}

func (r *imageAssetRepository) GetImageAssetForUser(ctx context.Context, userID int64, assetID string) (*service.ImageAsset, error) {
	if r == nil || r.sql == nil {
		return nil, errors.New("image asset database is unavailable")
	}
	asset := &service.ImageAsset{}
	err := r.sql.QueryRowContext(ctx, imageAssetSelectSQL+`
WHERE a.user_id = $1 AND a.asset_id = $2 AND a.deleted_at IS NULL`, userID, assetID).Scan(
		&asset.ID, &asset.TaskID, &asset.UserID, &asset.ImageIndex, &asset.ObjectKey,
		&asset.ContentType, &asset.ByteSize, &asset.CreatedAt, &asset.ExpiresAt,
		&asset.Prompt, &asset.Model, &asset.Mode, &asset.Size, &asset.Quality, &asset.OutputFormat)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrImageAssetNotFound
	}
	if err != nil {
		return nil, err
	}
	return asset, nil
}

func (r *imageAssetRepository) GetImageAssetForAPIKey(ctx context.Context, userID, apiKeyID int64, assetID string) (*service.ImageAsset, error) {
	if r == nil || r.sql == nil {
		return nil, errors.New("image asset database is unavailable")
	}
	asset := &service.ImageAsset{}
	err := r.sql.QueryRowContext(ctx, imageAssetSelectSQL+`
WHERE a.user_id = $1 AND j.api_key_id = $2 AND a.asset_id = $3 AND a.deleted_at IS NULL`, userID, apiKeyID, assetID).Scan(
		&asset.ID, &asset.TaskID, &asset.UserID, &asset.ImageIndex, &asset.ObjectKey,
		&asset.ContentType, &asset.ByteSize, &asset.CreatedAt, &asset.ExpiresAt,
		&asset.Prompt, &asset.Model, &asset.Mode, &asset.Size, &asset.Quality, &asset.OutputFormat)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrImageAssetNotFound
	}
	if err != nil {
		return nil, err
	}
	return asset, nil
}

func (r *imageAssetRepository) ListImageAssetsForTaskOwner(ctx context.Context, userID, apiKeyID int64, taskID string, now time.Time) ([]service.ImageAsset, error) {
	if r == nil || r.sql == nil {
		return nil, errors.New("image asset database is unavailable")
	}
	rows, err := r.sql.QueryContext(ctx, imageAssetSelectSQL+`
WHERE a.user_id = $1 AND j.api_key_id = $2 AND a.task_id = $3
  AND a.deleted_at IS NULL AND a.expires_at > $4
ORDER BY a.image_index ASC`, userID, apiKeyID, taskID, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanImageAssets(rows)
}

func (r *imageAssetRepository) DeleteImageAssetForUser(ctx context.Context, userID int64, assetID string) error {
	if r == nil || r.sql == nil {
		return errors.New("image asset database is unavailable")
	}
	result, err := r.sql.ExecContext(ctx, `
DELETE FROM image_assets WHERE asset_id = $1 AND user_id = $2 AND deleted_at IS NULL`, assetID, userID)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return service.ErrImageAssetNotFound
	}
	return nil
}

func (r *imageAssetRepository) ListExpiredImageAssets(ctx context.Context, now time.Time, limit int) ([]service.ImageAsset, error) {
	if r == nil || r.sql == nil {
		return nil, errors.New("image asset database is unavailable")
	}
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := r.sql.QueryContext(ctx, imageAssetSelectSQL+`
WHERE a.deleted_at IS NULL AND a.expires_at <= $1
ORDER BY a.expires_at ASC LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanImageAssets(rows)
}

func (r *imageAssetRepository) DeleteImageAssetByID(ctx context.Context, assetID string) error {
	if r == nil || r.sql == nil {
		return errors.New("image asset database is unavailable")
	}
	_, err := r.sql.ExecContext(ctx, `DELETE FROM image_assets WHERE asset_id = $1`, assetID)
	return err
}

func (r *imageAssetRepository) DeleteExpiredImageGenerationJobs(ctx context.Context, now time.Time) error {
	if r == nil || r.sql == nil {
		return errors.New("image asset database is unavailable")
	}
	_, err := r.sql.ExecContext(ctx, `
DELETE FROM image_generation_jobs j
WHERE j.expires_at <= $1
  AND NOT EXISTS (SELECT 1 FROM image_assets a WHERE a.task_id = j.task_id)`, now)
	return err
}

type imageAssetRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanImageAssets(rows imageAssetRows) ([]service.ImageAsset, error) {
	assets := make([]service.ImageAsset, 0)
	for rows.Next() {
		asset := service.ImageAsset{}
		if err := rows.Scan(
			&asset.ID, &asset.TaskID, &asset.UserID, &asset.ImageIndex, &asset.ObjectKey,
			&asset.ContentType, &asset.ByteSize, &asset.CreatedAt, &asset.ExpiresAt,
			&asset.Prompt, &asset.Model, &asset.Mode, &asset.Size, &asset.Quality, &asset.OutputFormat,
		); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return assets, nil
}
