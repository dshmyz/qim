package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestEnsureConversationAccessRejectsNonMember(t *testing.T) {
	db := setupHandlerTestDB(t)
	database.DB = db

	user := createTestUser(t, db)
	outsider := &model.User{Username: "outsider", PasswordHash: "hash", Nickname: "Outsider"}
	require.NoError(t, db.Create(outsider).Error)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID, Role: "member"}).Error)

	require.NoError(t, ensureConversationAccess(db, conv.ID, user.ID))
	require.ErrorIs(t, ensureConversationAccess(db, conv.ID, outsider.ID), errConversationAccessDenied)
}

func TestGenerateSummaryMetaRejectsNonMember(t *testing.T) {
	db := setupHandlerTestDB(t)
	database.DB = db

	owner := createTestUser(t, db)
	outsider := &model.User{Username: "summary-outsider", PasswordHash: "hash", Nickname: "Outsider"}
	require.NoError(t, db.Create(outsider).Error)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: owner.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.Message{ConversationID: conv.ID, SenderID: owner.ID, Type: "text", Content: "secret roadmap"}).Error)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", outsider.ID)
		c.Next()
	})
	r.POST("/summary/meta", (&AIHandler{}).GenerateSummaryMeta)

	body := bytes.NewBufferString(`{"conversation_id":` + fmt.Sprint(conv.ID) + `,"time_range":"7d"}`)
	req := httptest.NewRequest(http.MethodPost, "/summary/meta", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
}
