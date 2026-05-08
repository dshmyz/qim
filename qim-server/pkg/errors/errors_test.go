package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBusinessError_Error(t *testing.T) {
	err := NewBusinessError(1000, "test error")
	assert.Equal(t, "code=1000, message=test error", err.Error())
}

func TestPredefinedErrors(t *testing.T) {
	assert.Equal(t, 1000, ErrInternalError.Code)
	assert.Equal(t, 1001, ErrInvalidParams.Code)
	assert.Equal(t, 1002, ErrUnauthorized.Code)
	assert.Equal(t, 1003, ErrForbidden.Code)
	assert.Equal(t, 1004, ErrNotFound.Code)
}

func TestErrorCodes(t *testing.T) {
	assert.Equal(t, 0, ErrCodeSuccess)
	assert.Equal(t, 1000, ErrCodeInternalError)
	assert.Equal(t, 1001, ErrCodeInvalidParams)
	assert.Equal(t, 1002, ErrCodeUnauthorized)
	assert.Equal(t, 1003, ErrCodeForbidden)
	assert.Equal(t, 1004, ErrCodeNotFound)
	assert.Equal(t, 1005, ErrCodeConflict)
	assert.Equal(t, 1006, ErrCodeTooManyRequests)
}

func TestDomainErrorCodes(t *testing.T) {
	assert.Equal(t, 2000, ErrCodeUserNotFound)
	assert.Equal(t, 2001, ErrCodeUserAlreadyExists)
	assert.Equal(t, 2002, ErrCodeInvalidPassword)
	assert.Equal(t, 2003, ErrCodeUserDisabled)

	assert.Equal(t, 3000, ErrCodeConversationNotFound)
	assert.Equal(t, 3001, ErrCodeConversationForbidden)
	assert.Equal(t, 3002, ErrCodeNotMember)

	assert.Equal(t, 4000, ErrCodeMessageNotFound)
	assert.Equal(t, 4001, ErrCodeMessageForbidden)
	assert.Equal(t, 4002, ErrCodeMessageRecalled)

	assert.Equal(t, 5000, ErrCodeFileNotFound)
	assert.Equal(t, 5001, ErrCodeFileTooLarge)
	assert.Equal(t, 5002, ErrCodeFileUploadFailed)

	assert.Equal(t, 6000, ErrCodeGroupNotFound)
	assert.Equal(t, 6001, ErrCodeNotGroupOwner)
	assert.Equal(t, 6002, ErrCodeGroupFull)
}

func TestNewBusinessError(t *testing.T) {
	err := NewBusinessError(ErrCodeUserNotFound, "用户不存在")
	assert.Equal(t, ErrCodeUserNotFound, err.Code)
	assert.Equal(t, "用户不存在", err.Message)
}
