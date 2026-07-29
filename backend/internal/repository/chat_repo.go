package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type chatRepository struct{ db *sql.DB }

func NewChatRepository(db *sql.DB) service.ChatRepository { return &chatRepository{db: db} }

func (r *chatRepository) CreateConversation(ctx context.Context, userID int64, c *service.ChatConversation) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO chat_conversations
 (conversation_id,user_id,title,model,system_prompt,summary,version,created_at,updated_at)
 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, c.ID, userID, c.Title, c.Model, c.SystemPrompt, c.Summary, c.Version, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *chatRepository) ListConversations(ctx context.Context, userID int64, limit int) ([]service.ChatConversation, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT conversation_id,user_id,title,model,system_prompt,summary,version,created_at,updated_at
 FROM chat_conversations WHERE user_id=$1 ORDER BY updated_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.ChatConversation, 0)
	for rows.Next() {
		var c service.ChatConversation
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.Model, &c.SystemPrompt, &c.Summary, &c.Version, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *chatRepository) GetConversation(ctx context.Context, userID int64, id string) (*service.ChatConversation, error) {
	c := &service.ChatConversation{}
	err := r.db.QueryRowContext(ctx, `SELECT conversation_id,user_id,title,model,system_prompt,summary,version,created_at,updated_at
 FROM chat_conversations WHERE conversation_id=$1 AND user_id=$2`, id, userID).Scan(&c.ID, &c.UserID, &c.Title, &c.Model, &c.SystemPrompt, &c.Summary, &c.Version, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrChatNotFound
	}
	return c, err
}

func (r *chatRepository) UpdateConversation(ctx context.Context, userID int64, c *service.ChatConversation) error {
	result, err := r.db.ExecContext(ctx, `UPDATE chat_conversations SET title=$3,model=$4,system_prompt=$5,summary=$6,version=version+1,updated_at=$7
 WHERE conversation_id=$1 AND user_id=$2`, c.ID, userID, c.Title, c.Model, c.SystemPrompt, c.Summary, c.UpdatedAt)
	return chatAffected(result, err)
}
func (r *chatRepository) DeleteConversation(ctx context.Context, userID int64, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM chat_conversations WHERE conversation_id=$1 AND user_id=$2`, id, userID)
	return chatAffected(result, err)
}

func (r *chatRepository) ListMessages(ctx context.Context, userID int64, conversationID string, limit int) ([]service.ChatMessage, error) {
	if limit <= 0 || limit > 2000 {
		limit = 1000
	}
	rows, err := r.db.QueryContext(ctx, `SELECT message_id,conversation_id,user_id,parent_id,role,status,parts_json,token_count,created_at
 FROM chat_messages WHERE conversation_id=$1 AND user_id=$2 ORDER BY created_at,message_id LIMIT $3`, conversationID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.ChatMessage, 0)
	for rows.Next() {
		m, err := scanChatMessage(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, *m)
	}
	return items, rows.Err()
}

func (r *chatRepository) CreateMessage(ctx context.Context, m *service.ChatMessage) error {
	parts, err := json.Marshal(m.Parts)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO chat_messages
 (message_id,conversation_id,user_id,parent_id,role,status,parts_json,token_count,created_at) VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7::jsonb,$8,$9)`, m.ID, m.ConversationID, m.UserID, m.ParentID, m.Role, m.Status, string(parts), m.TokenCount, m.CreatedAt)
	return err
}

func (r *chatRepository) CreateRun(ctx context.Context, run *service.ChatRun) error {
	usage := run.Usage
	if len(usage) == 0 {
		usage = json.RawMessage(`{}`)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO chat_runs
 (run_id,conversation_id,user_id,user_message_id,assistant_message_id,status,idempotency_key,model,api_key_id,usage_json,error_message,created_at)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb,$11,$12)`, run.ID, run.ConversationID, run.UserID, run.UserMessageID, run.AssistantMessageID, run.Status, run.IdempotencyKey, run.Model, run.APIKeyID, string(usage), run.Error, run.CreatedAt)
	return err
}

func (r *chatRepository) GetRun(ctx context.Context, userID int64, id string) (*service.ChatRun, error) {
	return r.getRun(ctx, `WHERE run_id=$1 AND user_id=$2`, id, userID)
}
func (r *chatRepository) GetRunByIdempotency(ctx context.Context, userID int64, key string) (*service.ChatRun, error) {
	if key == "" {
		return nil, service.ErrChatNotFound
	}
	return r.getRun(ctx, `WHERE idempotency_key=$1 AND user_id=$2`, key, userID)
}
func (r *chatRepository) getRun(ctx context.Context, where string, args ...any) (*service.ChatRun, error) {
	query := `SELECT run_id,conversation_id,user_id,user_message_id,assistant_message_id,status,idempotency_key,model,api_key_id,usage_json,error_message,started_at,completed_at,created_at FROM chat_runs ` + where
	var run service.ChatRun
	var apiKeyID sql.NullInt64
	var usage []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&run.ID, &run.ConversationID, &run.UserID, &run.UserMessageID, &run.AssistantMessageID, &run.Status, &run.IdempotencyKey, &run.Model, &apiKeyID, &usage, &run.Error, &run.StartedAt, &run.CompletedAt, &run.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrChatNotFound
	}
	if err != nil {
		return nil, err
	}
	if apiKeyID.Valid {
		run.APIKeyID = &apiKeyID.Int64
	}
	run.Usage = json.RawMessage(append([]byte(nil), usage...))
	return &run, nil
}
func (r *chatRepository) SetRunRunning(ctx context.Context, userID int64, id string, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE chat_runs SET status='running',started_at=$3 WHERE run_id=$1 AND user_id=$2 AND status='queued'`, id, userID, at)
	return chatAffected(result, err)
}

func (r *chatRepository) FinishRun(ctx context.Context, userID int64, id, status, assistantText, errorText string, usage json.RawMessage, at time.Time) error {
	if len(usage) == 0 {
		usage = json.RawMessage(`{}`)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var messageID, conversationID string
	err = tx.QueryRowContext(ctx, `UPDATE chat_runs SET status=$3,usage_json=$4::jsonb,error_message=$5,completed_at=$6
 WHERE run_id=$1 AND user_id=$2 RETURNING assistant_message_id,conversation_id`, id, userID, status, string(usage), errorText, at).Scan(&messageID, &conversationID)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrChatNotFound
	}
	if err != nil {
		return err
	}
	messageStatus := service.ChatMessageCompleted
	if status == service.ChatRunFailed {
		messageStatus = service.ChatMessageFailed
	}
	if status == service.ChatRunCancelled {
		messageStatus = service.ChatMessageCancelled
	}
	parts, _ := json.Marshal([]service.ChatPart{{Type: "text", Text: assistantText}})
	if assistantText == "" {
		parts = []byte(`[]`)
	}
	result, err := tx.ExecContext(ctx, `UPDATE chat_messages SET status=$4,parts_json=$5::jsonb WHERE message_id=$1 AND conversation_id=$2 AND user_id=$3`, messageID, conversationID, userID, messageStatus, string(parts))
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return service.ErrChatNotFound
	}
	if _, err = tx.ExecContext(ctx, `UPDATE chat_conversations SET updated_at=$3,version=version+1 WHERE conversation_id=$1 AND user_id=$2`, conversationID, userID, at); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *chatRepository) CreateAttachment(ctx context.Context, a *service.ChatAttachment) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO chat_attachments
 (attachment_id,user_id,object_key,file_name,content_type,byte_size,sha256,created_at,expires_at)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, a.ID, a.UserID, a.ObjectKey, a.FileName, a.ContentType, a.ByteSize, a.SHA256, a.CreatedAt, a.ExpiresAt)
	return err
}
func (r *chatRepository) GetAttachment(ctx context.Context, userID int64, id string) (*service.ChatAttachment, error) {
	a := &service.ChatAttachment{}
	err := r.db.QueryRowContext(ctx, `SELECT attachment_id,user_id,object_key,file_name,content_type,byte_size,sha256,created_at,expires_at
 FROM chat_attachments WHERE attachment_id=$1 AND user_id=$2 AND expires_at>NOW()`, id, userID).Scan(&a.ID, &a.UserID, &a.ObjectKey, &a.FileName, &a.ContentType, &a.ByteSize, &a.SHA256, &a.CreatedAt, &a.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrChatAttachmentNotFound
	}
	return a, err
}
func (r *chatRepository) DeleteAttachment(ctx context.Context, userID int64, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM chat_attachments WHERE attachment_id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return service.ErrChatAttachmentNotFound
	}
	return nil
}

type chatScanner func(...any) error

func scanChatMessage(scan chatScanner) (*service.ChatMessage, error) {
	var m service.ChatMessage
	var parent sql.NullString
	var parts []byte
	if err := scan(&m.ID, &m.ConversationID, &m.UserID, &parent, &m.Role, &m.Status, &parts, &m.TokenCount, &m.CreatedAt); err != nil {
		return nil, err
	}
	if parent.Valid {
		m.ParentID = parent.String
	}
	if len(parts) > 0 {
		if err := json.Unmarshal(parts, &m.Parts); err != nil {
			return nil, fmt.Errorf("decode chat message parts: %w", err)
		}
	}
	if m.Parts == nil {
		m.Parts = []service.ChatPart{}
	}
	return &m, nil
}
func chatAffected(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, e := result.RowsAffected()
	if e == nil && n == 0 {
		return service.ErrChatNotFound
	}
	return e
}
