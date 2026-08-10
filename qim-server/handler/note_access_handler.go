package handler

import (
	"strconv"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetNoteAiAccessible 逐笔记切换「允许分身读取」状态。
// 打开 → 该笔记向量化进 user_notes_{userID} 集合（分身可检索到）；
// 关闭 → 移除该笔记向量（分身检索不到）。落库与向量同步见 NoteService.SetNoteAiAccessible。
//
// @Summary 切换笔记「允许分身读取」状态
// @Description 打开后笔记向量化进分身知识库集合，关闭后移除该笔记向量
// @Tags 笔记
// @Accept json
// @Produce json
// @Param id path int true "笔记ID"
// @Param request body object{ai_accessible=bool} true "可见性开关"
// @Success 200 {object} response.Response "更新后的笔记"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 404 {object} response.Response "笔记不存在"
// @Failure 500 {object} response.Response "服务器错误"
// @Router /api/v1/notes/{id}/access [patch]
func SetNoteAiAccessible(c *gin.Context) {
	userID, _ := c.Get("user_id")
	noteIDStr := c.Param("id")

	noteID, err := strconv.ParseUint(noteIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的笔记ID")
		return
	}

	var req struct {
		AiAccessible bool `json:"ai_accessible"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	noteSvc := di.GlobalContainer.NoteService
	note, err := noteSvc.SetNoteAiAccessible(userID.(uint), uint(noteID), req.AiAccessible)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFound(c, "笔记不存在")
			return
		}
		response.InternalServerError(c, "设置失败")
		return
	}

	response.Success(c, note)
}
