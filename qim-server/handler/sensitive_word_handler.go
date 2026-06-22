package handler

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/service"

	"github.com/gin-gonic/gin"
)

func GetSensitiveWords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	keyword := c.Query("keyword")

	swSvc := di.GlobalContainer.SensitiveWordService

	result, err := swSvc.GetSensitiveWords(service.SensitiveWordQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
	})
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	response.Success(c, gin.H{
		"list":     sensitiveWordsToFrontend(result.List),
		"total":    result.Total,
		"page":     result.Page,
		"pageSize": result.PageSize,
	})
}

func refreshSensitiveWordCache() {
	if di.GlobalContainer.MessageService != nil {
		di.GlobalContainer.MessageService.RefreshSensitiveWordCache()
	}
}

func CreateSensitiveWord(c *gin.Context) {
	var req struct {
		Word  string `json:"word" binding:"required"`
		Level string `json:"level"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	swSvc := di.GlobalContainer.SensitiveWordService

	if _, err := swSvc.GetByWord(req.Word); err == nil {
		response.BadRequest(c, "敏感词已存在")
		return
	}

	level := req.Level
	if level == "" {
		level = "medium"
	}

	word := model.SensitiveWord{
		Word:  req.Word,
		Level: level,
	}

	if err := swSvc.Create(&word); err != nil {
		response.InternalServerError(c, "创建失败")
		return
	}

	refreshSensitiveWordCache()
	response.Success(c, sensitiveWordToFrontend(word))
}

func UpdateSensitiveWord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		Word    string `json:"word"`
		Level   string `json:"level"`
		Enabled *bool  `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	swSvc := di.GlobalContainer.SensitiveWordService

	word, err := swSvc.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "敏感词不存在")
		return
	}

	if req.Word != "" {
		word.Word = req.Word
	}
	if req.Level != "" {
		word.Level = req.Level
	}
	if req.Enabled != nil {
		word.Enabled = *req.Enabled
	}

	if err := swSvc.Update(word); err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	refreshSensitiveWordCache()
	response.Success(c, sensitiveWordToFrontend(*word))
}

func DeleteSensitiveWord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	swSvc := di.GlobalContainer.SensitiveWordService

	if _, err := swSvc.GetByID(uint(id)); err != nil {
		response.NotFound(c, "敏感词不存在")
		return
	}

	if err := swSvc.Delete(uint(id)); err != nil {
		response.InternalServerError(c, "删除失败")
		return
	}

	refreshSensitiveWordCache()
	response.Success(c, gin.H{"message": "删除成功"})
}

func ToggleSensitiveWordStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	swSvc := di.GlobalContainer.SensitiveWordService

	word, err := swSvc.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "敏感词不存在")
		return
	}

	word.Enabled = !word.Enabled
	if err := swSvc.Update(word); err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	refreshSensitiveWordCache()
	response.Success(c, sensitiveWordToFrontend(*word))
}

func BatchCreateSensitiveWords(c *gin.Context) {
	var req struct {
		Words []string `json:"words" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	swSvc := di.GlobalContainer.SensitiveWordService
	count := 0

	for _, w := range req.Words {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}

		if _, err := swSvc.GetByWord(w); err == nil {
			continue
		}

		word := model.SensitiveWord{Word: w, Level: "medium"}
		if err := swSvc.Create(&word); err == nil {
			count++
		}
	}

	refreshSensitiveWordCache()
	response.Success(c, gin.H{"message": fmt.Sprintf("成功导入%d个敏感词", count), "count": count})
}

func ImportSensitiveWords(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "上传文件失败")
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		response.BadRequest(c, "CSV文件解析失败")
		return
	}

	swSvc := di.GlobalContainer.SensitiveWordService
	count := 0

	for i, record := range records {
		if i == 0 && len(record) > 0 && record[0] == "敏感词" {
			continue
		}
		if len(record) == 0 {
			continue
		}

		w := strings.TrimSpace(record[0])
		if w == "" {
			continue
		}

		if _, err := swSvc.GetByWord(w); err == nil {
			continue
		}

		word := model.SensitiveWord{Word: w, Level: "medium"}
		if err := swSvc.Create(&word); err == nil {
			count++
		}
	}

	refreshSensitiveWordCache()
	response.Success(c, gin.H{"message": fmt.Sprintf("成功导入%d个敏感词", count), "count": count})
}

func ExportSensitiveWords(c *gin.Context) {
	swSvc := di.GlobalContainer.SensitiveWordService

	words, err := swSvc.GetAll()
	if err != nil {
		response.InternalServerError(c, "导出失败")
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=sensitive_words.csv")

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"敏感词", "级别", "状态"})

	for _, word := range words {
		enabled := "启用"
		if !word.Enabled {
			enabled = "停用"
		}
		w.Write([]string{word.Word, word.Level, enabled})
	}
	w.Flush()
}

func CheckSensitiveWords(c *gin.Context) {
	var req struct {
		Text string `json:"text" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	swSvc := di.GlobalContainer.SensitiveWordService

	words, err := swSvc.GetAllEnabled()
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	found := []string{}
	for _, word := range words {
		if strings.Contains(req.Text, word.Word) {
			found = append(found, word.Word)
		}
	}

	response.Success(c, gin.H{
		"contains_sensitive": len(found) > 0,
		"words":              found,
	})
}

func RegisterSensitiveWordRoutes(r *gin.RouterGroup) {
	r.GET("/sensitive-words", GetSensitiveWords)
	r.POST("/sensitive-words", CreateSensitiveWord)
	r.PUT("/sensitive-words/:id", UpdateSensitiveWord)
	r.DELETE("/sensitive-words/:id", DeleteSensitiveWord)
	r.PATCH("/sensitive-words/:id/toggle", ToggleSensitiveWordStatus)
	r.POST("/sensitive-words/batch", BatchCreateSensitiveWords)
	r.POST("/sensitive-words/import", ImportSensitiveWords)
	r.GET("/sensitive-words/export", ExportSensitiveWords)
	r.POST("/sensitive-words/check", CheckSensitiveWords)
}
