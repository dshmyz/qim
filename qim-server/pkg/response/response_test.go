package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dshmyz/qim/qim-server/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestSuccess(t *testing.T) {
	c, w := setupTestContext()
	Success(c, gin.H{"key": "value"})

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, errors.ErrCodeSuccess, resp.Code)
	assert.Equal(t, "success", resp.Message)
}

func TestSuccessWithMessage(t *testing.T) {
	c, w := setupTestContext()
	SuccessWithMessage(c, "操作成功", gin.H{"id": 1})

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, errors.ErrCodeSuccess, resp.Code)
	assert.Equal(t, "操作成功", resp.Message)
}

func TestBadRequest(t *testing.T) {
	c, w := setupTestContext()
	BadRequest(c, "invalid param")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, errors.ErrCodeInvalidParams, resp.Code)
	assert.Equal(t, "invalid param", resp.Message)
}

func TestUnauthorized(t *testing.T) {
	c, w := setupTestContext()
	Unauthorized(c, "未授权")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, errors.ErrCodeUnauthorized, resp.Code)
}

func TestForbidden(t *testing.T) {
	c, w := setupTestContext()
	Forbidden(c, "无权限")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, errors.ErrCodeForbidden, resp.Code)
}

func TestNotFound(t *testing.T) {
	c, w := setupTestContext()
	NotFound(c, "资源不存在")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, errors.ErrCodeNotFound, resp.Code)
}

func TestConflict(t *testing.T) {
	c, w := setupTestContext()
	Conflict(c, "资源冲突")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, errors.ErrCodeConflict, resp.Code)
}

func TestInternalServerError(t *testing.T) {
	c, w := setupTestContext()
	InternalServerError(c, "服务器错误")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, errors.ErrCodeInternalError, resp.Code)
}

func TestTooManyRequests(t *testing.T) {
	c, w := setupTestContext()
	TooManyRequests(c, "请求过于频繁")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, errors.ErrCodeTooManyRequests, resp.Code)
}

func TestFromBusinessError(t *testing.T) {
	c, w := setupTestContext()
	err := errors.NewBusinessError(errors.ErrCodeNotFound, "user not found")
	FromBusinessError(c, err)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, errors.ErrCodeNotFound, resp.Code)
	assert.Equal(t, "user not found", resp.Message)
}

func TestFromBusinessError_DefaultToInternalError(t *testing.T) {
	c, w := setupTestContext()
	err := errors.NewBusinessError(errors.ErrCodeUserNotFound, "用户不存在")
	FromBusinessError(c, err)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, errors.ErrCodeUserNotFound, resp.Code)
}

func TestErrorWithDetail(t *testing.T) {
	c, w := setupTestContext()
	ErrorWithDetail(c, http.StatusBadRequest, errors.ErrCodeInvalidParams, "参数错误", gin.H{"field": "username"})

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, float64(errors.ErrCodeInvalidParams), resp["code"])
	assert.Equal(t, "参数错误", resp["message"])
}
