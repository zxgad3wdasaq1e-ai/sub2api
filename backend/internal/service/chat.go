package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/google/uuid"
)

const (
	ChatRunQueued                = "queued"
	ChatRunRunning               = "running"
	ChatRunCompleted             = "completed"
	ChatRunFailed                = "failed"
	ChatRunCancelled             = "cancelled"
	ChatMessagePending           = "pending"
	ChatMessageCompleted         = "completed"
	ChatMessageFailed            = "failed"
	ChatMessageCancelled         = "cancelled"
	maxChatAttachmentBytes int64 = 32 << 20
)

var (
	ErrChatNotFound              = infraerrors.NotFound("CHAT_NOT_FOUND", "chat resource not found")
	ErrChatUnavailable           = infraerrors.ServiceUnavailable("CHAT_UNAVAILABLE", "chat service is unavailable")
	ErrChatInvalid               = infraerrors.BadRequest("CHAT_INVALID", "invalid chat request")
	ErrChatAttachmentNotFound    = infraerrors.NotFound("CHAT_ATTACHMENT_NOT_FOUND", "chat attachment not found")
	ErrChatAttachmentUnavailable = infraerrors.ServiceUnavailable("CHAT_ATTACHMENT_UNAVAILABLE", "chat attachment storage is unavailable")
)

var chatTemperatureUnsupportedModel = regexp.MustCompile(`^(o\d|gpt-5)`)

type ChatConversation struct {
	ID           string    `json:"id"`
	UserID       int64     `json:"-"`
	Title        string    `json:"title"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
	Summary      string    `json:"summary,omitempty"`
	Version      int64     `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ChatPart struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	AttachmentID string `json:"attachment_id,omitempty"`
	Name         string `json:"name,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
	ByteSize     int64  `json:"byte_size,omitempty"`
	PreviewURL   string `json:"preview_url,omitempty"`
}

type ChatMessage struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	UserID         int64      `json:"-"`
	ParentID       string     `json:"parent_id,omitempty"`
	Role           string     `json:"role"`
	Status         string     `json:"status"`
	Parts          []ChatPart `json:"parts"`
	TokenCount     int        `json:"token_count,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ChatRun struct {
	ID                 string          `json:"id"`
	ConversationID     string          `json:"conversation_id"`
	UserID             int64           `json:"-"`
	UserMessageID      string          `json:"user_message_id"`
	AssistantMessageID string          `json:"assistant_message_id"`
	Status             string          `json:"status"`
	IdempotencyKey     string          `json:"-"`
	Model              string          `json:"model"`
	APIKeyID           *int64          `json:"api_key_id,omitempty"`
	Usage              json.RawMessage `json:"usage,omitempty"`
	Error              string          `json:"error,omitempty"`
	StartedAt          *time.Time      `json:"started_at,omitempty"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

type ChatAttachment struct {
	ID          string    `json:"id"`
	UserID      int64     `json:"-"`
	ObjectKey   string    `json:"-"`
	FileName    string    `json:"name"`
	ContentType string    `json:"content_type"`
	ByteSize    int64     `json:"byte_size"`
	SHA256      string    `json:"sha256"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type ChatKeyOption struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	GroupID *int64 `json:"group_id"`
	Status  string `json:"status"`
}

type CreateConversationInput struct{ Title, Model, SystemPrompt string }
type CreateChatRunInput struct {
	Content        string
	Attachments    []ChatPart
	Model          string
	SystemPrompt   *string
	Temperature    *float64
	APIKeyID       *int64
	IdempotencyKey string
}

type ChatRepository interface {
	CreateConversation(context.Context, int64, *ChatConversation) error
	ListConversations(context.Context, int64, int) ([]ChatConversation, error)
	GetConversation(context.Context, int64, string) (*ChatConversation, error)
	UpdateConversation(context.Context, int64, *ChatConversation) error
	DeleteConversation(context.Context, int64, string) error
	ListMessages(context.Context, int64, string, int) ([]ChatMessage, error)
	CreateMessage(context.Context, *ChatMessage) error
	CreateRun(context.Context, *ChatRun) error
	GetRun(context.Context, int64, string) (*ChatRun, error)
	GetRunByIdempotency(context.Context, int64, string) (*ChatRun, error)
	SetRunRunning(context.Context, int64, string, time.Time) error
	FinishRun(context.Context, int64, string, string, string, string, json.RawMessage, time.Time) error
	CreateAttachment(context.Context, *ChatAttachment) error
	GetAttachment(context.Context, int64, string) (*ChatAttachment, error)
	DeleteAttachment(context.Context, int64, string) error
}

type ChatService struct {
	repo    ChatRepository
	apiKeys *APIKeyService
	resolve ImageStorageResolver
}

func NewChatService(repo ChatRepository, apiKeys *APIKeyService, settings *ImageStorageSettingService) *ChatService {
	var resolve ImageStorageResolver
	if settings != nil {
		resolve = settings.Resolver()
	}
	return &ChatService{repo: repo, apiKeys: apiKeys, resolve: resolve}
}

func (s *ChatService) CreateConversation(ctx context.Context, userID int64, in CreateConversationInput) (*ChatConversation, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrChatUnavailable
	}
	now := time.Now().UTC()
	conversation := &ChatConversation{ID: "conv_" + strings.ReplaceAll(uuid.NewString(), "-", ""), UserID: userID,
		Title: strings.TrimSpace(in.Title), Model: strings.TrimSpace(in.Model), SystemPrompt: in.SystemPrompt,
		Version: 1, CreatedAt: now, UpdatedAt: now}
	if conversation.Title == "" {
		conversation.Title = "New conversation"
	}
	if err := s.repo.CreateConversation(ctx, userID, conversation); err != nil {
		return nil, err
	}
	return conversation, nil
}

func (s *ChatService) ListConversations(ctx context.Context, userID int64) ([]ChatConversation, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrChatUnavailable
	}
	return s.repo.ListConversations(ctx, userID, 100)
}

func (s *ChatService) ListKeyOptions(ctx context.Context, userID int64) ([]ChatKeyOption, error) {
	if s == nil || s.apiKeys == nil || userID <= 0 {
		return nil, ErrChatUnavailable
	}
	keys, _, err := s.apiKeys.List(ctx, userID, defaultAPIKeyPagination(), APIKeyListFilters{Status: StatusActive})
	if err != nil {
		return nil, err
	}
	items := make([]ChatKeyOption, 0, len(keys))
	for _, key := range keys {
		items = append(items, ChatKeyOption{ID: key.ID, Name: key.Name, GroupID: key.GroupID, Status: key.Status})
	}
	return items, nil
}
func (s *ChatService) GetConversation(ctx context.Context, userID int64, id string) (*ChatConversation, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrChatUnavailable
	}
	return s.repo.GetConversation(ctx, userID, strings.TrimSpace(id))
}
func (s *ChatService) DeleteConversation(ctx context.Context, userID int64, id string) error {
	if s == nil || s.repo == nil || userID <= 0 {
		return ErrChatUnavailable
	}
	return s.repo.DeleteConversation(ctx, userID, strings.TrimSpace(id))
}
func (s *ChatService) ListMessages(ctx context.Context, userID int64, id string) ([]ChatMessage, error) {
	if _, err := s.GetConversation(ctx, userID, id); err != nil {
		return nil, err
	}
	return s.repo.ListMessages(ctx, userID, id, 1000)
}

func (s *ChatService) UploadAttachment(ctx context.Context, userID int64, fileName, contentType string, data []byte) (*ChatAttachment, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrChatUnavailable
	}
	if len(data) == 0 || int64(len(data)) > maxChatAttachmentBytes {
		return nil, ErrChatInvalid
	}
	contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if !strings.HasPrefix(contentType, "image/") {
		return nil, ErrChatInvalid
	}
	detected := strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0])
	if detected != "image/png" && detected != "image/jpeg" && detected != "image/webp" {
		return nil, ErrChatInvalid
	}
	contentType = detected
	id := "att_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	key := fmt.Sprintf("chat/%d/%s", userID, id)
	hash := sha256.Sum256(data)
	attachment := &ChatAttachment{ID: id, UserID: userID, ObjectKey: key, FileName: strings.TrimSpace(fileName), ContentType: contentType,
		ByteSize: int64(len(data)), SHA256: hex.EncodeToString(hash[:]), CreatedAt: time.Now().UTC()}
	attachment.ExpiresAt = attachment.CreatedAt.Add(30 * 24 * time.Hour)
	uploader := s.currentUploader()
	if uploader == nil {
		return nil, ErrChatAttachmentUnavailable
	}
	if err := uploader.SaveObject(ctx, key, contentType, data); err != nil {
		return nil, ErrChatAttachmentUnavailable
	}
	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		_ = uploader.Delete(ctx, key)
		return nil, err
	}
	return attachment, nil
}

func (s *ChatService) currentUploader() *ImageResultUploader {
	if s == nil || s.resolve == nil {
		return nil
	}
	u, enabled := s.resolve()
	if !enabled {
		return nil
	}
	return u
}
func (s *ChatService) OpenAttachment(ctx context.Context, userID int64, id string) (*ChatAttachment, io.ReadCloser, error) {
	attachment, err := s.repo.GetAttachment(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	uploader := s.currentUploader()
	if uploader == nil {
		return nil, nil, ErrChatAttachmentUnavailable
	}
	body, _, _, err := uploader.Open(ctx, attachment.ObjectKey)
	if err != nil {
		return nil, nil, ErrChatAttachmentUnavailable
	}
	return attachment, body, nil
}

func (s *ChatService) CreateRun(ctx context.Context, userID int64, conversationID string, in CreateChatRunInput) (*ChatRun, error) {
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	if key := strings.TrimSpace(in.IdempotencyKey); key != "" {
		if existing, e := s.repo.GetRunByIdempotency(ctx, userID, key); e == nil && existing != nil {
			return existing, nil
		}
	}
	model := strings.TrimSpace(in.Model)
	if model == "" {
		model = conversation.Model
	}
	if model == "" {
		return nil, ErrChatInvalid
	}
	if in.SystemPrompt != nil {
		conversation.SystemPrompt = *in.SystemPrompt
	}
	var apiKeyID *int64 = in.APIKeyID
	var apiKey *APIKey
	if s.apiKeys != nil && apiKeyID != nil && *apiKeyID > 0 {
		apiKey, err = s.apiKeys.GetByID(ctx, *apiKeyID)
	} else if s.apiKeys != nil {
		keys, _, e := s.apiKeys.List(ctx, userID, defaultAPIKeyPagination(), APIKeyListFilters{Status: StatusActive})
		if e == nil && len(keys) > 0 {
			apiKey = &keys[0]
		} else {
			err = e
		}
	}
	if err != nil || apiKey == nil || apiKey.UserID != userID || !apiKey.IsActive() || apiKey.Group == nil || apiKey.IsExpired() || apiKey.IsQuotaExhausted() {
		return nil, ErrChatInvalid
	}
	if apiKeyID == nil {
		id := apiKey.ID
		apiKeyID = &id
	}
	parts := make([]ChatPart, 0, len(in.Attachments)+1)
	for _, requested := range in.Attachments {
		attachment, e := s.repo.GetAttachment(ctx, userID, strings.TrimSpace(requested.AttachmentID))
		if e != nil {
			return nil, ErrChatAttachmentNotFound
		}
		parts = append(parts, ChatPart{Type: "image", AttachmentID: attachment.ID, Name: attachment.FileName,
			ContentType: attachment.ContentType, ByteSize: attachment.ByteSize, PreviewURL: "/api/v1/chat/attachments/" + attachment.ID + "/content"})
	}
	if strings.TrimSpace(in.Content) != "" {
		parts = append([]ChatPart{{Type: "text", Text: in.Content}}, parts...)
	}
	if len(parts) == 0 {
		return nil, ErrChatInvalid
	}
	userMessage := &ChatMessage{ID: "msg_" + strings.ReplaceAll(uuid.NewString(), "-", ""), ConversationID: conversationID, UserID: userID, Role: "user", Status: ChatMessageCompleted, Parts: parts, CreatedAt: time.Now().UTC()}
	assistantMessage := &ChatMessage{ID: "msg_" + strings.ReplaceAll(uuid.NewString(), "-", ""), ConversationID: conversationID, UserID: userID, Role: "assistant", Status: ChatMessagePending, Parts: []ChatPart{}, CreatedAt: time.Now().UTC()}
	if err := s.repo.CreateMessage(ctx, userMessage); err != nil {
		return nil, err
	}
	if err := s.repo.CreateMessage(ctx, assistantMessage); err != nil {
		return nil, err
	}
	run := &ChatRun{ID: "run_" + strings.ReplaceAll(uuid.NewString(), "-", ""), ConversationID: conversationID, UserID: userID, UserMessageID: userMessage.ID, AssistantMessageID: assistantMessage.ID,
		Status: ChatRunQueued, IdempotencyKey: strings.TrimSpace(in.IdempotencyKey), Model: model, APIKeyID: apiKeyID, Usage: json.RawMessage(`{}`), CreatedAt: time.Now().UTC()}
	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	conversation.Model, conversation.UpdatedAt = model, time.Now().UTC()
	if strings.TrimSpace(conversation.Title) == "New conversation" {
		if in.Content != "" {
			conversation.Title = trimTitle(in.Content)
		}
	}
	_ = s.repo.UpdateConversation(ctx, userID, conversation)
	return run, nil
}

func trimTitle(v string) string {
	v = strings.Join(strings.Fields(v), " ")
	if len(v) > 42 {
		return v[:42]
	}
	if v == "" {
		return "New conversation"
	}
	return v
}
func defaultAPIKeyPagination() pagination.PaginationParams {
	return pagination.PaginationParams{Page: 1, PageSize: 100}
}

func (s *ChatService) BuildGatewayBody(ctx context.Context, userID int64, runID string, temperature *float64) ([]byte, *APIKey, error) {
	run, err := s.repo.GetRun(ctx, userID, runID)
	if err != nil {
		return nil, nil, err
	}
	conversation, err := s.GetConversation(ctx, userID, run.ConversationID)
	if err != nil {
		return nil, nil, err
	}
	messages, err := s.repo.ListMessages(ctx, userID, run.ConversationID, 1000)
	if err != nil {
		return nil, nil, err
	}
	if run.APIKeyID == nil {
		return nil, nil, ErrChatInvalid
	}
	apiKey, err := s.apiKeys.GetByID(ctx, *run.APIKeyID)
	if err != nil || apiKey == nil || apiKey.UserID != userID || !apiKey.IsActive() || apiKey.Group == nil {
		return nil, nil, ErrChatInvalid
	}
	requestMessages := make([]map[string]any, 0, len(messages)+1)
	if strings.TrimSpace(conversation.SystemPrompt) != "" {
		requestMessages = append(requestMessages, map[string]any{"role": "system", "content": conversation.SystemPrompt})
	}
	for _, message := range messages {
		// The assistant row for the current run is created as pending before the
		// gateway call. It is a persistence placeholder, not a context turn.
		if message.ID == run.AssistantMessageID || message.Status == ChatMessagePending || message.Status == ChatMessageCancelled || message.Status == ChatMessageFailed {
			continue
		}
		if message.Role == "user" {
			content := make([]map[string]any, 0, len(message.Parts))
			for _, part := range message.Parts {
				if part.Type == "text" {
					content = append(content, map[string]any{"type": "text", "text": part.Text})
					continue
				}
				if part.Type != "image" || part.AttachmentID == "" {
					continue
				}
				attachment, e := s.repo.GetAttachment(ctx, userID, part.AttachmentID)
				if e != nil {
					return nil, nil, ErrChatAttachmentNotFound
				}
				uploader := s.currentUploader()
				if uploader == nil {
					return nil, nil, ErrChatAttachmentUnavailable
				}
				body, _, _, e := uploader.Open(ctx, attachment.ObjectKey)
				if e != nil {
					return nil, nil, ErrChatAttachmentUnavailable
				}
				data, e := io.ReadAll(io.LimitReader(body, maxChatAttachmentBytes+1))
				_ = body.Close()
				if e != nil || int64(len(data)) > maxChatAttachmentBytes {
					return nil, nil, ErrChatAttachmentUnavailable
				}
				content = append(content, map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:" + attachment.ContentType + ";base64," + base64.StdEncoding.EncodeToString(data)}})
			}
			if len(content) == 1 && content[0]["type"] == "text" {
				requestMessages = append(requestMessages, map[string]any{"role": message.Role, "content": content[0]["text"]})
			} else {
				requestMessages = append(requestMessages, map[string]any{"role": message.Role, "content": content})
			}
		} else {
			text := ""
			for _, part := range message.Parts {
				if part.Type == "text" {
					text += part.Text
				}
			}
			requestMessages = append(requestMessages, map[string]any{"role": message.Role, "content": text})
		}
	}
	body := map[string]any{"model": run.Model, "messages": requestMessages, "stream": true}
	if temperature != nil && *temperature >= 0 && !chatTemperatureUnsupportedModel.MatchString(strings.ToLower(run.Model)) {
		body["temperature"] = *temperature
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	return encoded, apiKey, nil
}
func (s *ChatService) MarkRunning(ctx context.Context, userID int64, runID string) error {
	return s.repo.SetRunRunning(ctx, userID, runID, time.Now().UTC())
}
func (s *ChatService) GetRun(ctx context.Context, userID int64, id string) (*ChatRun, error) {
	return s.repo.GetRun(ctx, userID, id)
}
func (s *ChatService) FinishRun(ctx context.Context, userID int64, runID, status, assistantText, errText string, usage json.RawMessage) error {
	return s.repo.FinishRun(ctx, userID, runID, status, assistantText, errText, usage, time.Now().UTC())
}
func (s *ChatService) DeleteAttachment(ctx context.Context, userID int64, id string) error {
	a, err := s.repo.GetAttachment(ctx, userID, id)
	if err != nil {
		return err
	}
	if u := s.currentUploader(); u != nil {
		_ = u.Delete(ctx, a.ObjectKey)
	}
	return s.repo.DeleteAttachment(ctx, userID, id)
}
