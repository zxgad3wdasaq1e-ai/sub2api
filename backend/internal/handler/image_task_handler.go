package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AsyncImageHandler struct {
	tasks   *service.ImageTaskService
	assets  *service.ImageAssetService
	openAI  *OpenAIGatewayHandler
	execute func(platform string, c *gin.Context)
}

func NewAsyncImageHandler(tasks *service.ImageTaskService, assets *service.ImageAssetService, openAI *OpenAIGatewayHandler) *AsyncImageHandler {
	h := &AsyncImageHandler{tasks: tasks, assets: assets, openAI: openAI}
	h.execute = h.executeWithGateway
	return h
}

// enabled reports whether the async image task feature is available. Object
// storage is the enablement gate: without it the endpoints are fully disabled
// so that large base64 results never land in Redis.
func (h *AsyncImageHandler) enabled() bool {
	return h != nil && h.tasks != nil && h.tasks.Enabled()
}

// pollable reports whether task lookups can be served. It is deliberately weaker
// than enabled(): results already written to Redis stay readable after the
// feature is switched off, so an in-flight task is never stranded.
func (h *AsyncImageHandler) pollable() bool {
	return h != nil && h.tasks != nil && h.tasks.Pollable()
}

// Submit accepts the same payload as the synchronous Images endpoint and
// returns before the upstream image generation begins.
func (h *AsyncImageHandler) Submit(c *gin.Context) {
	if !h.enabled() {
		imageTaskJSONError(c, http.StatusNotFound, "not_found_error", "async image tasks are not enabled")
		return
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.UserID <= 0 || apiKey.ID <= 0 {
		imageTaskError(c, service.ErrImageTaskForbidden)
		return
	}
	platform := ""
	if apiKey.Group != nil {
		platform = apiKey.Group.Platform
	}
	if platform != service.PlatformOpenAI && platform != service.PlatformGrok {
		imageTaskJSONError(c, http.StatusNotFound, "not_found_error", "Images API is not supported for this platform")
		return
	}
	if !service.GroupAllowsImageGeneration(apiKey.Group) {
		imageTaskJSONError(c, http.StatusForbidden, "permission_error", service.ImageGenerationPermissionMessage())
		return
	}
	if h == nil || h.tasks == nil || h.execute == nil {
		imageTaskError(c, service.ErrImageTaskUnavailable)
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			imageTaskJSONError(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		imageTaskJSONError(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		imageTaskJSONError(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}
	if asyncImageRequestStreams(c.GetHeader("Content-Type"), body) {
		imageTaskJSONError(c, http.StatusBadRequest, "invalid_request_error", "streaming image requests cannot be submitted as asynchronous tasks")
		return
	}
	if err := h.validateRequest(c, platform, body); err != nil {
		imageTaskJSONError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	if !h.checkSecurityAuditBeforeSubmit(c, apiKey, platform, body) {
		return
	}

	metadata, err := h.taskMetadata(c, platform, body)
	if err != nil {
		imageTaskJSONError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	taskCtx, recorder, cancel := newAsyncImageContext(c, body, h.tasks.ExecutionTimeout())
	task, err := h.tasks.Create(c.Request.Context(), service.ImageTaskOwner{UserID: apiKey.UserID, APIKeyID: apiKey.ID}, metadata)
	if err != nil {
		cancel()
		imageTaskError(c, err)
		return
	}

	pollURL := imageTaskPollURL(c.Request.URL.Path, task.ID)
	c.Header("Cache-Control", "no-store")
	c.Header("Location", pollURL)
	c.Header("Retry-After", "3")
	c.JSON(http.StatusAccepted, gin.H{
		"id":         task.ID,
		"task_id":    task.TaskID,
		"object":     task.Object,
		"status":     task.Status,
		"created_at": task.CreatedAt,
		"expires_at": task.ExpiresAt,
		"poll_url":   pollURL,
	})

	go h.run(task.ID, platform, taskCtx, recorder, cancel)
}

// ListAssets returns the current user's durable image library. Object bytes
// are always served through this application, never from a browser-visible
// object-storage URL.
func (h *AsyncImageHandler) ListAssets(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 || h == nil || h.assets == nil {
		imageTaskError(c, service.ErrImageAssetUnavailable)
		return
	}
	assets, err := h.assets.List(c.Request.Context(), subject.UserID)
	if err != nil {
		imageTaskError(c, err)
		return
	}
	for i := range assets {
		assets[i].URL = imageAssetUserContentURL(assets[i].ID)
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": assets})
}

// GetAssetContent streams one current-user asset from internal object storage.
func (h *AsyncImageHandler) GetAssetContent(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 || h == nil || h.assets == nil {
		imageTaskError(c, service.ErrImageAssetUnavailable)
		return
	}
	asset, body, err := h.assets.Open(c.Request.Context(), subject.UserID, c.Param("asset_id"))
	if err != nil {
		imageTaskError(c, err)
		return
	}
	h.streamAssetContent(c, asset, body)
}

// GetAssetContentByAPIKey streams one asset owned by the current API key. It
// lets Images API clients fetch task result URLs without exposing MinIO.
func (h *AsyncImageHandler) GetAssetContentByAPIKey(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.UserID <= 0 || apiKey.ID <= 0 || h == nil || h.assets == nil {
		imageTaskError(c, service.ErrImageTaskForbidden)
		return
	}
	asset, body, err := h.assets.OpenForAPIKey(c.Request.Context(), apiKey.UserID, apiKey.ID, c.Param("asset_id"))
	if err != nil {
		imageTaskError(c, err)
		return
	}
	h.streamAssetContent(c, asset, body)
}

// DeleteAsset removes both the object and the current user's metadata record.
func (h *AsyncImageHandler) DeleteAsset(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 || h == nil || h.assets == nil {
		imageTaskError(c, service.ErrImageAssetUnavailable)
		return
	}
	if err := h.assets.Delete(c.Request.Context(), subject.UserID, c.Param("asset_id")); err != nil {
		imageTaskError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AsyncImageHandler) checkSecurityAuditBeforeSubmit(c *gin.Context, apiKey *service.APIKey, platform string, body []byte) bool {
	if h == nil || h.openAI == nil {
		return true
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		imageTaskJSONError(c, http.StatusInternalServerError, "api_error", "User context not found")
		return false
	}
	model := ""
	moderationBody := body
	if platform == service.PlatformGrok {
		parsed := service.ParseGrokMediaRequest(c.GetHeader("Content-Type"), body)
		model, moderationBody = parsed.Model, parsed.ModerationBody()
	} else if h.openAI.gatewayService != nil {
		parsed, err := h.openAI.gatewayService.ParseOpenAIImagesRequest(c, body)
		if err != nil {
			imageTaskJSONError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
			return false
		}
		model, moderationBody = parsed.Model, parsed.ModerationBody()
	}
	if len(moderationBody) == 0 {
		c.Set(securityAuditCompletedContextKey, true)
		return true
	}
	reqLog := requestLogger(c, "handler.async_image.security_audit",
		zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", apiKey.ID), zap.String("model", model))
	decision := h.openAI.checkSecurityAudit(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, model, moderationBody)
	if decision != nil && !decision.AllowNextStage {
		h.openAI.openAISecurityAuditError(c, decision)
		return false
	}
	return true
}

func (h *AsyncImageHandler) Get(c *gin.Context) {
	// Polling deliberately does not require the feature to be enabled, only that
	// the task store is reachable. Turning the switch off in the admin UI must not
	// strand tasks that were already accepted — their results are still in Redis
	// and their submitters are still polling.
	if !h.pollable() {
		imageTaskJSONError(c, http.StatusNotFound, "not_found_error", "async image tasks are not enabled")
		return
	}
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.UserID <= 0 || apiKey.ID <= 0 {
		imageTaskError(c, service.ErrImageTaskForbidden)
		return
	}
	task, err := h.tasks.Get(c.Request.Context(), service.ImageTaskOwner{UserID: apiKey.UserID, APIKeyID: apiKey.ID}, c.Param("task_id"))
	if err != nil {
		imageTaskError(c, err)
		return
	}
	if task.Status == service.ImageTaskStatusCompleted && h.assets != nil {
		assets, err := h.assets.ListForTaskOwner(c.Request.Context(), apiKey.UserID, apiKey.ID, task.ID)
		if err != nil {
			imageTaskError(c, err)
			return
		}
		rewriteImageTaskAssetURLs(task, assets, c.Request.URL.Path)
	}
	c.Header("Cache-Control", "no-store")
	if task.Status == service.ImageTaskStatusProcessing {
		c.Header("Retry-After", "3")
	}
	c.JSON(http.StatusOK, task)
}

func (h *AsyncImageHandler) validateRequest(c *gin.Context, platform string, body []byte) error {
	if h.openAI == nil || h.openAI.gatewayService == nil {
		return nil
	}
	if platform == service.PlatformGrok {
		parsed := service.ParseGrokMediaRequest(c.GetHeader("Content-Type"), body)
		if strings.TrimSpace(parsed.Model) == "" {
			return errors.New("model is required")
		}
		return nil
	}
	parsed, err := h.openAI.gatewayService.ParseOpenAIImagesRequest(c, body)
	if err != nil {
		return err
	}
	if parsed.Stream {
		return errors.New("streaming image requests cannot be submitted as asynchronous tasks")
	}
	return nil
}

func (h *AsyncImageHandler) taskMetadata(c *gin.Context, platform string, body []byte) (service.ImageTaskMetadata, error) {
	metadata := service.ImageTaskMetadata{Mode: "text"}
	if strings.Contains(c.Request.URL.Path, "/edits") {
		metadata.Mode = "edit"
	}
	if platform == service.PlatformGrok {
		parsed := service.ParseGrokMediaRequest(c.GetHeader("Content-Type"), body)
		metadata.Prompt = parsed.Prompt
		metadata.Model = parsed.Model
		metadata.Size = parsed.Size
		var options struct {
			Quality      string `json:"quality"`
			OutputFormat string `json:"output_format"`
		}
		_ = json.Unmarshal(body, &options)
		metadata.Quality = strings.TrimSpace(options.Quality)
		metadata.OutputFormat = strings.TrimSpace(options.OutputFormat)
		return metadata, nil
	}
	if h == nil || h.openAI == nil || h.openAI.gatewayService == nil {
		return metadata, errors.New("image gateway is unavailable")
	}
	parsed, err := h.openAI.gatewayService.ParseOpenAIImagesRequest(c, body)
	if err != nil {
		return metadata, err
	}
	metadata.Prompt = parsed.Prompt
	metadata.Model = parsed.Model
	metadata.Size = parsed.Size
	metadata.Quality = parsed.Quality
	metadata.OutputFormat = parsed.OutputFormat
	return metadata, nil
}

func (h *AsyncImageHandler) executeWithGateway(platform string, c *gin.Context) {
	if h.openAI == nil {
		imageTaskJSONError(c, http.StatusServiceUnavailable, "api_error", "image gateway is unavailable")
		return
	}
	if platform == service.PlatformGrok {
		h.openAI.GrokImages(c)
		return
	}
	h.openAI.Images(c)
}

func (h *AsyncImageHandler) run(taskID, platform string, taskCtx *gin.Context, recorder *httptest.ResponseRecorder, cancel context.CancelFunc) {
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().Error("image_task.execution_panicked", zap.String("task_id", taskID), zap.Any("panic", recovered))
			h.failTask(taskID, http.StatusInternalServerError, imageTaskErrorPayload("api_error", "image generation task panicked"))
		}
	}()

	h.execute(platform, taskCtx)
	body := bytes.TrimSpace(recorder.Body.Bytes())
	if err := taskCtx.Request.Context().Err(); err != nil && len(body) == 0 {
		h.failTask(taskID, http.StatusGatewayTimeout, imageTaskErrorPayload("timeout_error", "image generation task timed out"))
		return
	}
	statusCode := recorder.Code
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
		if len(body) == 0 || !json.Valid(body) {
			h.failTask(taskID, http.StatusBadGateway, imageTaskErrorPayload("api_error", "upstream returned an invalid image response"))
			return
		}
		if err := h.tasks.Complete(context.Background(), taskID, statusCode, json.RawMessage(body)); err != nil {
			logger.L().Error("image_task.complete_store_failed", zap.String("task_id", taskID), zap.Error(err))
		}
		return
	}
	h.failTask(taskID, statusCode, extractImageTaskError(body))
}

func (h *AsyncImageHandler) failTask(taskID string, statusCode int, taskErr json.RawMessage) {
	if err := h.tasks.Fail(context.Background(), taskID, statusCode, taskErr); err != nil {
		logger.L().Error("image_task.failure_store_failed", zap.String("task_id", taskID), zap.Error(err))
	}
}

func newAsyncImageContext(c *gin.Context, body []byte, timeoutDuration time.Duration) (*gin.Context, *httptest.ResponseRecorder, context.CancelFunc) {
	base := context.WithoutCancel(c.Request.Context())
	executionCtx, cancel := context.WithTimeout(base, timeoutDuration)
	request := c.Request.Clone(executionCtx)
	request.Body = io.NopCloser(bytes.NewReader(body))
	request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	request.ContentLength = int64(len(body))
	request.URL.Path = strings.TrimSuffix(request.URL.Path, "/async")

	taskCtx := c.Copy()
	recorder := httptest.NewRecorder()
	recorderCtx, _ := gin.CreateTestContext(recorder)
	taskCtx.Writer = recorderCtx.Writer
	taskCtx.Request = request
	return taskCtx, recorder, cancel
}

func asyncImageRequestStreams(contentType string, body []byte) bool {
	if isMultipartImagesContentType(contentType) {
		return false
	}
	var envelope struct {
		Stream bool `json:"stream"`
	}
	return json.Unmarshal(body, &envelope) == nil && envelope.Stream
}

func imageTaskPollURL(submitPath, taskID string) string {
	if strings.HasPrefix(submitPath, "/v1/") {
		return "/v1/images/tasks/" + taskID
	}
	return "/images/tasks/" + taskID
}

func imageAssetUserContentURL(assetID string) string {
	return "/user/image-assets/" + url.PathEscape(assetID) + "/content"
}

func imageTaskAssetContentURL(requestPath, assetID string) string {
	prefix := "/images/assets/"
	if strings.HasPrefix(requestPath, "/v1/") {
		prefix = "/v1/images/assets/"
	}
	return prefix + url.PathEscape(assetID) + "/content"
}

func rewriteImageTaskAssetURLs(task *service.ImageTask, assets []service.ImageAsset, requestPath string) {
	if task == nil || len(assets) == 0 {
		return
	}
	assetURLs := make(map[int]string, len(assets))
	for _, asset := range assets {
		assetURLs[asset.ImageIndex] = imageTaskAssetContentURL(requestPath, asset.ID)
	}
	if firstURL, ok := assetURLs[assets[0].ImageIndex]; ok {
		task.ImageURL = firstURL
	}
	if !json.Valid(task.Result) {
		return
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(task.Result, &envelope); err != nil {
		return
	}
	rawData, ok := envelope["data"]
	if !ok {
		return
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(rawData, &items); err != nil {
		return
	}
	for index, item := range items {
		assetURL, ok := assetURLs[index]
		if !ok {
			continue
		}
		encodedURL, err := json.Marshal(assetURL)
		if err != nil {
			return
		}
		item["url"] = encodedURL
		items[index] = item
	}
	encodedItems, err := json.Marshal(items)
	if err != nil {
		return
	}
	envelope["data"] = encodedItems
	encodedResult, err := json.Marshal(envelope)
	if err == nil {
		task.Result = encodedResult
	}
}

func (h *AsyncImageHandler) streamAssetContent(c *gin.Context, asset *service.ImageAsset, body io.ReadCloser) {
	defer func() { _ = body.Close() }()
	contentType := strings.TrimSpace(strings.Split(asset.ContentType, ";")[0])
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		contentType = "application/octet-stream"
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Type", contentType)
	if asset.ByteSize > 0 {
		c.Header("Content-Length", strconv.FormatInt(asset.ByteSize, 10))
	}
	if c.Query("download") == "1" {
		extension := "png"
		switch {
		case strings.Contains(contentType, "jpeg"):
			extension = "jpg"
		case strings.Contains(contentType, "webp"):
			extension = "webp"
		case strings.Contains(contentType, "gif"):
			extension = "gif"
		}
		c.Header("Content-Disposition", "attachment; filename=\"sub2api-"+asset.ID+"."+extension+"\"")
	}
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, body); err != nil {
		logger.L().Warn("image_asset.proxy_stream_failed", zap.String("asset_id", asset.ID), zap.Error(err))
	}
}

func extractImageTaskError(body []byte) json.RawMessage {
	if json.Valid(body) {
		var envelope struct {
			Error json.RawMessage `json:"error"`
		}
		if json.Unmarshal(body, &envelope) == nil && len(envelope.Error) > 0 && json.Valid(envelope.Error) {
			return envelope.Error
		}
		return json.RawMessage(body)
	}
	return imageTaskErrorPayload("api_error", "image generation failed")
}

func imageTaskErrorPayload(errorType, message string) json.RawMessage {
	data, _ := json.Marshal(gin.H{"type": errorType, "message": message})
	return data
}

func imageTaskError(c *gin.Context, err error) {
	status := infraerrors.Code(err)
	code := infraerrors.Reason(err)
	message := infraerrors.Message(err)
	if status <= 0 {
		status = http.StatusInternalServerError
	}
	if strings.TrimSpace(code) == "" {
		code = "IMAGE_TASK_ERROR"
	}
	imageTaskJSONError(c, status, code, message)
}

func imageTaskJSONError(c *gin.Context, status int, code, message string) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, gin.H{"error": gin.H{"type": code, "code": code, "message": message}})
}
