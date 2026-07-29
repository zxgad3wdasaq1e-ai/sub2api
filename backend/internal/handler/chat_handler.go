package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	chat          *service.ChatService
	openAI        *OpenAIGatewayHandler
	gateway       *GatewayHandler
	subscriptions *service.SubscriptionService
	mu            sync.Mutex
	cancellations map[string]context.CancelFunc
}

func NewChatHandler(chat *service.ChatService, openAI *OpenAIGatewayHandler, gateway *GatewayHandler, subscriptions *service.SubscriptionService) *ChatHandler {
	return &ChatHandler{chat: chat, openAI: openAI, gateway: gateway, subscriptions: subscriptions, cancellations: make(map[string]context.CancelFunc)}
}

func chatSubject(c *gin.Context) (middleware2.AuthSubject, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	return subject, ok && subject.UserID > 0
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	items, err := h.chat.ListConversations(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *ChatHandler) ListKeyOptions(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	items, err := h.chat.ListKeyOptions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type createConversationRequest struct {
	Title        string `json:"title"`
	Model        string `json:"model"`
	SystemPrompt string `json:"system_prompt"`
}

func (h *ChatHandler) CreateConversation(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.ErrorFrom(c, service.ErrChatInvalid)
		return
	}
	conversation, err := h.chat.CreateConversation(c.Request.Context(), subject.UserID, service.CreateConversationInput{Title: req.Title, Model: req.Model, SystemPrompt: req.SystemPrompt})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.JSON(http.StatusCreated, conversation)
}
func (h *ChatHandler) GetConversation(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	conversation, err := h.chat.GetConversation(c.Request.Context(), subject.UserID, c.Param("conversation_id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	messages, err := h.chat.ListMessages(c.Request.Context(), subject.UserID, conversation.ID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"conversation": conversation, "messages": messages})
}
func (h *ChatHandler) GetMessages(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	messages, err := h.chat.ListMessages(c.Request.Context(), subject.UserID, c.Param("conversation_id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": messages})
}
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if err := h.chat.DeleteConversation(c.Request.Context(), subject.UserID, c.Param("conversation_id")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type createRunRequest struct {
	Content        string   `json:"content"`
	Model          string   `json:"model"`
	SystemPrompt   *string  `json:"system_prompt"`
	Temperature    *float64 `json:"temperature"`
	APIKeyID       *int64   `json:"api_key_id"`
	IdempotencyKey string   `json:"idempotency_key"`
	Attachments    []struct {
		AttachmentID string `json:"attachment_id"`
	} `json:"attachments"`
}

func (h *ChatHandler) CreateRun(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req createRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrChatInvalid)
		return
	}
	parts := make([]service.ChatPart, 0, len(req.Attachments))
	for _, a := range req.Attachments {
		parts = append(parts, service.ChatPart{Type: "image", AttachmentID: a.AttachmentID})
	}
	run, err := h.chat.CreateRun(c.Request.Context(), subject.UserID, c.Param("conversation_id"), service.CreateChatRunInput{Content: req.Content, Attachments: parts, Model: req.Model, SystemPrompt: req.SystemPrompt, Temperature: req.Temperature, APIKeyID: req.APIKeyID, IdempotencyKey: req.IdempotencyKey})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// A duplicate idempotency request can safely resume the existing run.
	if run.Status != service.ChatRunQueued {
		h.streamExistingRun(c, subject, run, req.Temperature)
		return
	}
	h.streamRun(c, subject, run, req.Temperature)
}

func (h *ChatHandler) GetRun(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	run, err := h.chat.GetRun(c.Request.Context(), subject.UserID, c.Param("run_id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, run)
}

func (h *ChatHandler) GetRunEvents(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	run, err := h.chat.GetRun(c.Request.Context(), subject.UserID, c.Param("run_id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-store")
	c.SSEvent("run", run)
	c.SSEvent("done", gin.H{"status": run.Status})
	c.Writer.Flush()
}

func (h *ChatHandler) CancelRun(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id := c.Param("run_id")
	if _, err := h.chat.GetRun(c.Request.Context(), subject.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.mu.Lock()
	cancel := h.cancellations[id]
	h.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	c.Status(http.StatusAccepted)
}

func (h *ChatHandler) UploadAttachment(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 40<<20)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.ErrorFrom(c, service.ErrChatInvalid)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 32<<20+1))
	if err != nil || len(data) == 0 || len(data) > 32<<20 {
		response.ErrorFrom(c, service.ErrChatInvalid)
		return
	}
	attachment, err := h.chat.UploadAttachment(c.Request.Context(), subject.UserID, header.Filename, header.Header.Get("Content-Type"), data)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.JSON(http.StatusCreated, attachment)
}
func (h *ChatHandler) GetAttachmentContent(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	attachment, body, err := h.chat.OpenAttachment(c.Request.Context(), subject.UserID, c.Param("attachment_id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer body.Close()
	c.Header("Cache-Control", "private, max-age=300")
	c.DataFromReader(http.StatusOK, attachment.ByteSize, attachment.ContentType, body, nil)
}
func (h *ChatHandler) DeleteAttachment(c *gin.Context) {
	subject, ok := chatSubject(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if err := h.chat.DeleteAttachment(c.Request.Context(), subject.UserID, c.Param("attachment_id")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ChatHandler) streamExistingRun(c *gin.Context, subject middleware2.AuthSubject, run *service.ChatRun, temp *float64) {
	if run.Status == service.ChatRunCompleted || run.Status == service.ChatRunFailed || run.Status == service.ChatRunCancelled {
		c.Header("X-Chat-Run-ID", run.ID)
		c.JSON(http.StatusOK, run)
		return
	}
	h.streamRun(c, subject, run, temp)
}

func (h *ChatHandler) streamRun(c *gin.Context, subject middleware2.AuthSubject, run *service.ChatRun, temp *float64) {
	userID := subject.UserID
	body, apiKey, err := h.chat.BuildGatewayBody(c.Request.Context(), userID, run.ID, temp)
	if err != nil {
		_ = h.chat.FinishRun(c.Request.Context(), userID, run.ID, service.ChatRunFailed, "", err.Error(), nil)
		response.ErrorFrom(c, err)
		return
	}
	if err := h.chat.MarkRunning(c.Request.Context(), userID, run.ID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	h.mu.Lock()
	h.cancellations[run.ID] = cancel
	h.mu.Unlock()
	defer func() { h.mu.Lock(); delete(h.cancellations, run.ID); h.mu.Unlock() }()
	c.Request = c.Request.WithContext(ctx)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Accept", "text/event-stream")
	c.Request.URL.Path = "/v1/chat/completions"
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeyUser), subject)
	if h.subscriptions != nil && apiKey.Group != nil && apiKey.GroupID != nil {
		if sub, e := h.subscriptions.GetActiveSubscription(ctx, userID, *apiKey.GroupID); e == nil && sub != nil {
			c.Set(string(middleware2.ContextKeySubscription), sub)
		}
	}
	ensureCompositeTargetPlatform(c, apiKey, run.Model)
	c.Header("X-Chat-Run-ID", run.ID)
	c.Header("X-Chat-User-Message-ID", run.UserMessageID)
	c.Header("X-Chat-Assistant-Message-ID", run.AssistantMessageID)
	c.Header("Cache-Control", "no-store")
	c.Header("X-Accel-Buffering", "no")
	capture := &chatCaptureWriter{ResponseWriter: c.Writer}
	c.Writer = capture
	platform := ""
	if apiKey.Group != nil {
		platform = apiKey.Group.Platform
		if platform == service.PlatformComposite {
			if resolved, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context()); ok {
				platform = resolved
			}
		}
	}
	if platform == service.PlatformOpenAI || platform == service.PlatformGrok {
		if h.openAI == nil {
			err = service.ErrChatUnavailable
		} else {
			h.openAI.ChatCompletions(c)
		}
	} else if h.gateway == nil {
		err = service.ErrChatUnavailable
	} else {
		h.gateway.ChatCompletions(c)
	}
	status := capture.Status()
	if status == 0 {
		status = http.StatusOK
	}
	text, usage, upstreamErr := parseChatGatewayResponse(capture.Bytes())
	if err != nil {
		upstreamErr = err.Error()
	}
	finalStatus := service.ChatRunCompleted
	errorText := ""
	if ctx.Err() != nil {
		finalStatus = service.ChatRunCancelled
		errorText = "generation cancelled"
	} else if status >= http.StatusBadRequest || upstreamErr != "" {
		finalStatus = service.ChatRunFailed
		errorText = upstreamErr
		if errorText == "" {
			errorText = "upstream request failed"
		}
	}
	if e := h.chat.FinishRun(context.Background(), userID, run.ID, finalStatus, text, errorText, usage); e != nil { /* response has already been committed; keep the run error in logs */
	}
}

type chatCaptureWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *chatCaptureWriter) Write(b []byte) (int, error) {
	_, _ = w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}
func (w *chatCaptureWriter) WriteString(s string) (int, error) {
	_, _ = w.buf.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
func (w *chatCaptureWriter) Bytes() []byte { return w.buf.Bytes() }

func parseChatGatewayResponse(body []byte) (string, json.RawMessage, string) {
	var text string
	var usage json.RawMessage
	var upstreamErr string
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 4096), 16<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var root map[string]any
		if json.Unmarshal([]byte(data), &root) != nil {
			continue
		}
		if u, ok := root["usage"]; ok {
			if raw, e := json.Marshal(u); e == nil {
				usage = raw
			}
		}
		if e, ok := root["error"].(map[string]any); ok {
			if msg, _ := e["message"].(string); msg != "" {
				upstreamErr = msg
			}
		}
		if typ, _ := root["type"].(string); typ == "response.output_text.delta" {
			if delta, _ := root["delta"].(string); delta != "" {
				text += delta
			}
		} else if delta, ok := root["delta"].(string); ok {
			text += delta
		}
		if choices, ok := root["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok {
				if d, ok := choice["delta"].(map[string]any); ok {
					if value, _ := d["content"].(string); value != "" {
						text += value
					}
				}
				if m, ok := choice["message"].(map[string]any); ok {
					if value, _ := m["content"].(string); value != "" {
						text += value
					}
				}
			}
		}
	}
	if text == "" && !bytes.HasPrefix(bytes.TrimSpace(body), []byte("data:")) {
		var root map[string]any
		if json.Unmarshal(body, &root) == nil {
			if choices, ok := root["choices"].([]any); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]any); ok {
					if message, ok := choice["message"].(map[string]any); ok {
						if value, _ := message["content"].(string); value != "" {
							text = value
						}
					}
				}
			}
			if e, ok := root["error"].(map[string]any); ok {
				upstreamErr, _ = e["message"].(string)
			}
		}
	}
	return text, usage, upstreamErr
}
