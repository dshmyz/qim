package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/pkg/upload"
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

// refreshSensitiveWordCache 刷新敏感词缓存，失败时记录日志并返回 error。
// 历史问题：原先静默调用且不返回错误，CRUD 成功但缓存刷新失败时管理员无感知，
// 新增/修改的敏感词不会生效，造成缓存与 DB 不一致。
func refreshSensitiveWordCache() error {
	if di.GlobalContainer.MessageService == nil {
		return nil
	}
	if err := di.GlobalContainer.MessageService.RefreshSensitiveWordCache(); err != nil {
		logger.WithModule("SensitiveWord").Error("刷新敏感词缓存失败，CRUD 已成功但缓存可能不一致", "error", err)
		return err
	}
	return nil
}

// respondWithCacheRefresh 在敏感词 CRUD 成功后刷新缓存并返回响应。
// 缓存刷新失败时不改变成功状态（数据已落库），但通过 message 透出告警，
// 让管理员感知过滤可能暂时不一致，便于重试或排查。
// 统一封装避免 6 个 handler 重复处理。
func respondWithCacheRefresh(c *gin.Context, data interface{}) {
	if err := refreshSensitiveWordCache(); err != nil {
		response.SuccessWithMessage(c, "操作已保存，但缓存刷新失败，敏感词过滤可能暂时不一致，请稍后重试", data)
		return
	}
	response.Success(c, data)
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

	respondWithCacheRefresh(c, sensitiveWordToFrontend(word))
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

	respondWithCacheRefresh(c, sensitiveWordToFrontend(*word))
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

	respondWithCacheRefresh(c, gin.H{"message": "删除成功"})
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

	respondWithCacheRefresh(c, sensitiveWordToFrontend(*word))
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

	respondWithCacheRefresh(c, gin.H{"message": fmt.Sprintf("成功导入%d个敏感词", count), "count": count})
}

// importPolicy 是 CSV 导入专用的上传策略：仅允许 .csv，大小限制 10MB。
// 复用 upload.Policy 统一校验逻辑，避免散落的自定义校验。
var importPolicy = upload.NewPolicy(
	upload.DefaultImportMaxSize,
	map[string]bool{".csv": true},
	true, // 启用白名单校验
)

func ImportSensitiveWords(c *gin.Context) {
	// MaxBytesReader 保护：防止超大 multipart 请求在解析阶段就耗尽内存
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, importPolicy.MaxSize+64*1024)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "上传文件失败")
		return
	}
	defer file.Close()

	// 统一走 upload.Policy 校验：大小 + 类型（黑名单 + 白名单）
	if err := importPolicy.ValidateSize(header.Size); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := importPolicy.ValidateType(header.Filename, ""); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 读取文件头检测真实 MIME，防止伪装扩展名（.csv 实为 .exe）
	headBuf := make([]byte, 512)
	n, _ := file.Read(headBuf)
	headBuf = headBuf[:n]
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		response.BadRequest(c, "读取文件失败")
		return
	}
	detectedMime := upload.DetectMimeType(headBuf)
	if err := importPolicy.ValidateType(header.Filename, detectedMime); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 限制读取字节数，防止实际内容超过 header 声称的大小
	limitedReader := io.LimitReader(file, importPolicy.MaxSize+1)
	reader := csv.NewReader(limitedReader)
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

	respondWithCacheRefresh(c, gin.H{"message": fmt.Sprintf("成功导入%d个敏感词", count), "count": count})
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

	// 走 MessageService 的缓存路径，避免每次直接查库
	msgSvc := di.GlobalContainer.MessageService
	contains, _ := msgSvc.CheckSensitiveContent(req.Text)

	// 普通用户接口仅返回是否包含敏感词，不返回命中词列表（防止词库被枚举还原）
	response.Success(c, gin.H{
		"contains_sensitive": contains,
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
}
