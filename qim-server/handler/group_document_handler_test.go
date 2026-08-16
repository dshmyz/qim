package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupAddGroupDocumentTest 构造 AddGroupDocument 白名单测试环境：
// 内存 sqlite + 群主身份管理员，返回路由与种子文件创建 helper。
// :id 路由参数是会话 ID（conversation_id），非群 ID，与 requireGroupAdmin 的查询对齐。
func setupAddGroupDocumentTest(t *testing.T) (*gin.Engine, *gorm.DB, uint, func(string, string) uint) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.Group{},
		&model.File{},
		&model.GroupDocument{},
	))
	database.DB = db

	admin := model.User{Username: "admin", PasswordHash: "hash", Nickname: "管理员", Status: "offline"}
	require.NoError(t, db.Create(&admin).Error)

	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	group := model.Group{ConversationID: conv.ID, GroupType: "group", Name: "测试群", CreatorID: admin.ID}
	require.NoError(t, db.Create(&group).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: admin.ID, Role: "owner"}).Error)

	createFile := func(name, mime string) uint {
		f := model.File{
			UserID: admin.ID, ScopeType: "user", Name: name, OriginalName: name,
			Size: 10, MimeType: mime, StoragePath: "uploads/" + name,
		}
		require.NoError(t, db.Create(&f).Error)
		return f.ID
	}

	r := gin.New()
	r.POST("/groups/:id/documents", func(c *gin.Context) {
		c.Set("user_id", admin.ID)
		c.Next()
	}, AddGroupDocument)

	return r, db, conv.ID, createFile
}

func addGroupDocumentRequest(router *gin.Engine, convID, fileID uint) *httptest.ResponseRecorder {
	body := bytes.NewBufferString(`{"file_id":` + uintToStr(fileID) + `}`)
	req := httptest.NewRequest(http.MethodPost, "/groups/"+uintToStr(convID)+"/documents", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestAddGroupDocument_ImageMimeAllowed 验证直接上传的图片（MIME 命中 image/*）
// 可绑定进群知识库，供后续视觉 OCR 入库。
func TestAddGroupDocument_ImageMimeAllowed(t *testing.T) {
	r, db, convID, createFile := setupAddGroupDocumentTest(t)
	fileID := createFile("screenshot.png", "image/png")

	w := addGroupDocumentRequest(r, convID, fileID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "文档绑定成功")

	var doc model.GroupDocument
	require.NoError(t, db.Where("group_id = ? AND file_id = ?", 1, fileID).First(&doc).Error, "图片绑定后应产生群文档记录")
}

// TestAddGroupDocument_ImageExtFallback 验证 MIME 异常（如 OOXML 的 application/zip 场景）
// 的图片仍可按原始扩展名兜底识别并绑定。
func TestAddGroupDocument_ImageExtFallback(t *testing.T) {
	r, _, convID, createFile := setupAddGroupDocumentTest(t)
	fileID := createFile("photo.jpg", "application/octet-stream")

	w := addGroupDocumentRequest(r, convID, fileID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// TestAddGroupDocument_NonImageRejected 验证非文档/非图片类型（如视频）仍被白名单拒绝，
// 不因图片放行而误放其他类型。
func TestAddGroupDocument_NonImageRejected(t *testing.T) {
	r, _, convID, createFile := setupAddGroupDocumentTest(t)
	fileID := createFile("clip.mp4", "video/mp4")

	w := addGroupDocumentRequest(r, convID, fileID)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "只支持添加文档或图片")
}
