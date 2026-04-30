package handler

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func GetSensitiveWords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")

	db := database.GetDB()

	query := db.Model(&model.SensitiveWord{})
	if keyword != "" {
		query = query.Where("word LIKE ?", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var words []model.SensitiveWord
	offset := (page - 1) * pageSize
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&words)

	response.Success(c, gin.H{
		"list":     words,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
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

	db := database.GetDB()

	var existing model.SensitiveWord
	if err := db.Where("word = ? AND deleted_at IS NULL", req.Word).First(&existing).Error; err == nil {
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

	if err := db.Create(&word).Error; err != nil {
		response.InternalServerError(c, "创建失败")
		return
	}

	response.Success(c, word)
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

	db := database.GetDB()
	var word model.SensitiveWord
	if err := db.First(&word, uint(id)).Error; err != nil {
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

	if err := db.Save(&word).Error; err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	response.Success(c, word)
}

func DeleteSensitiveWord(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	db := database.GetDB()
	var word model.SensitiveWord
	if err := db.First(&word, uint(id)).Error; err != nil {
		response.NotFound(c, "敏感词不存在")
		return
	}

	if err := db.Delete(&word).Error; err != nil {
		response.InternalServerError(c, "删除失败")
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

func ToggleSensitiveWordStatus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	db := database.GetDB()
	var word model.SensitiveWord
	if err := db.First(&word, uint(id)).Error; err != nil {
		response.NotFound(c, "敏感词不存在")
		return
	}

	word.Enabled = !word.Enabled
	if err := db.Save(&word).Error; err != nil {
		response.InternalServerError(c, "更新失败")
		return
	}

	response.Success(c, word)
}

func BatchCreateSensitiveWords(c *gin.Context) {
	var req struct {
		Words []string `json:"words" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()
	count := 0

	for _, w := range req.Words {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}

		var existing model.SensitiveWord
		if err := db.Where("word = ? AND deleted_at IS NULL", w).First(&existing).Error; err == nil {
			continue
		}

		word := model.SensitiveWord{Word: w, Level: "medium"}
		if err := db.Create(&word).Error; err == nil {
			count++
		}
	}

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

	db := database.GetDB()
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

		var existing model.SensitiveWord
		if err := db.Where("word = ? AND deleted_at IS NULL", w).First(&existing).Error; err == nil {
			continue
		}

		word := model.SensitiveWord{Word: w, Level: "medium"}
		if err := db.Create(&word).Error; err == nil {
			count++
		}
	}

	response.Success(c, gin.H{"message": fmt.Sprintf("成功导入%d个敏感词", count), "count": count})
}

func ExportSensitiveWords(c *gin.Context) {
	db := database.GetDB()
	var words []model.SensitiveWord
	db.Find(&words)

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

	db := database.GetDB()
	var words []model.SensitiveWord
	db.Where("enabled = ?", true).Find(&words)

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
