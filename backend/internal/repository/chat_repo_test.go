package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestChatRepositoryConversationOwnershipIsolation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	repo := NewChatRepository(db)
	query := regexp.QuoteMeta(`SELECT conversation_id,user_id,title,model,system_prompt,summary,version,created_at,updated_at
 FROM chat_conversations WHERE conversation_id=$1 AND user_id=$2`)

	// User A cannot discover user B's conversation, even with the exact ID.
	mock.ExpectQuery(query).WithArgs("conv_b", int64(101)).WillReturnRows(sqlmock.NewRows([]string{
		"conversation_id", "user_id", "title", "model", "system_prompt", "summary", "version", "created_at", "updated_at",
	}))
	_, err = repo.GetConversation(context.Background(), 101, "conv_b")
	require.ErrorIs(t, err, service.ErrChatNotFound)

	now := time.Now().UTC()
	mock.ExpectQuery(query).WithArgs("conv_b", int64(202)).WillReturnRows(sqlmock.NewRows([]string{
		"conversation_id", "user_id", "title", "model", "system_prompt", "summary", "version", "created_at", "updated_at",
	}).AddRow("conv_b", int64(202), "B conversation", "gpt-5", "", "", int64(1), now, now))
	conversation, err := repo.GetConversation(context.Background(), 202, "conv_b")
	require.NoError(t, err)
	require.Equal(t, int64(202), conversation.UserID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatRepositoryRunAndAttachmentOwnershipIsolation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	repo := NewChatRepository(db)

	runQuery := regexp.QuoteMeta(`SELECT run_id,conversation_id,user_id,user_message_id,assistant_message_id,status,idempotency_key,model,api_key_id,usage_json,error_message,started_at,completed_at,created_at FROM chat_runs WHERE run_id=$1 AND user_id=$2`)
	mock.ExpectQuery(runQuery).WithArgs("run_b", int64(101)).WillReturnRows(sqlmock.NewRows([]string{
		"run_id", "conversation_id", "user_id", "user_message_id", "assistant_message_id", "status", "idempotency_key", "model", "api_key_id", "usage_json", "error_message", "started_at", "completed_at", "created_at",
	}))
	_, err = repo.GetRun(context.Background(), 101, "run_b")
	require.ErrorIs(t, err, service.ErrChatNotFound)

	attachmentQuery := regexp.QuoteMeta(`SELECT attachment_id,user_id,object_key,file_name,content_type,byte_size,sha256,created_at,expires_at
 FROM chat_attachments WHERE attachment_id=$1 AND user_id=$2 AND expires_at>NOW()`)
	mock.ExpectQuery(attachmentQuery).WithArgs("att_b", int64(101)).WillReturnRows(sqlmock.NewRows([]string{
		"attachment_id", "user_id", "object_key", "file_name", "content_type", "byte_size", "sha256", "created_at", "expires_at",
	}))
	_, err = repo.GetAttachment(context.Background(), 101, "att_b")
	require.True(t, errors.Is(err, service.ErrChatAttachmentNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatRepositoryRunStateTransitions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})
	repo := NewChatRepository(db)
	now := time.Now().UTC()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE chat_runs SET status='running',started_at=$3 WHERE run_id=$1 AND user_id=$2 AND status='queued'`)).
		WithArgs("run_1", int64(101), now).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.SetRunRunning(context.Background(), 101, "run_1", now))

	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE chat_runs SET status=\\$3").
		WithArgs("run_1", int64(101), service.ChatRunCompleted, `{}`, "", now).
		WillReturnRows(sqlmock.NewRows([]string{"assistant_message_id", "conversation_id"}).AddRow("msg_a", "conv_1"))
	mock.ExpectExec("UPDATE chat_messages SET status=\\$4").
		WithArgs("msg_a", "conv_1", int64(101), service.ChatMessageCompleted, `[{"type":"text","text":"done"}]`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE chat_conversations SET updated_at=\\$3").
		WithArgs("conv_1", int64(101), now).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	require.NoError(t, repo.FinishRun(context.Background(), 101, "run_1", service.ChatRunCompleted, "done", "", nil, now))
	require.NoError(t, mock.ExpectationsWereMet())
}
