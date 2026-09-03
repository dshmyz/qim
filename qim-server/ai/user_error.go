package ai

import "errors"

// UserFacingError 携带面向用户的文案 + 底层原因。
// GetCompletion 统一把完成调用错误包成该类型：Error() 保留原始错误（日志/排查用），
// UserMessage 面向前端展示。这样分类只发生在 GetCompletion 一处，调用点统一取
// ai.UserMessage(err) 即可，不必各自重复猜测「连接失败还是内容问题」。
type UserFacingError struct {
	UserMessage string
	Err         error
}

func (e *UserFacingError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.UserMessage
}

func (e *UserFacingError) Unwrap() error { return e.Err }

// WrapCompletionError 把完成调用错误包装为 UserFacingError（带分类文案 + 原始原因）。
// err 为空时返回 nil，避免调用方误判。
func WrapCompletionError(err error) error {
	if err == nil {
		return nil
	}
	return &UserFacingError{UserMessage: ClassifyCompletionError(err), Err: err}
}

// UserMessage 从任意错误中提取面向用户的文案：UserFacingError 返回分类结果，
// 其余（含未包装的裸错误）返回通用兜底，保证前端永远拿到可读文案。
func UserMessage(err error) string {
	if err == nil {
		return ""
	}
	var ue *UserFacingError
	if errors.As(err, &ue) && ue.UserMessage != "" {
		return ue.UserMessage
	}
	return "AI 服务暂时不可用，请稍后重试"
}
