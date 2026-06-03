package handler

import (
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

type NoteVectorSearchRequest struct {
	Query string `json:"query" binding:"required"`
	TopK  int    `json:"top_k"`
}

type NoteVectorSearchItem struct {
	Content string  `json:"content"`
	Score   float64 `json:"score"`
	Title   string  `json:"title"`
	NoteID  string  `json:"note_id"`
}

func NoteVectorSearch(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req NoteVectorSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.TopK <= 0 || req.TopK > 20 {
		req.TopK = 5
	}

	noteVectorSvc := di.GlobalContainer.NoteVectorService
	if noteVectorSvc == nil {
		response.InternalServerError(c, "向量检索服务不可用")
		return
	}

	results, err := noteVectorSvc.SearchNotes(userID.(uint), req.Query, req.TopK)
	if err != nil {
		response.InternalServerError(c, "搜索失败: "+err.Error())
		return
	}

	items := make([]NoteVectorSearchItem, 0, len(results))
	for _, r := range results {
		items = append(items, NoteVectorSearchItem{
			Content: r.Content,
			Score:   r.Score,
			Title:   r.Metadata["title"],
			NoteID:  r.Metadata["note_id"],
		})
	}

	response.Success(c, gin.H{
		"results": items,
		"total":   len(items),
	})
}
