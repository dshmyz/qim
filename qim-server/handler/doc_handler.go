package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/gin-gonic/gin"
)

// docSlugToFilename 映射 URL slug 到 docs/ 目录下的文件名。
var docSlugToFilename = map[string]string{
	"cli":  "CLI使用指南.md",
	"mcp":  "MCP接入指南.md",
}

// GetDocContent 读取 docs/ 目录下的 Markdown 文件并返回内容。
// @Summary      获取开发文档内容
// @Description  根据 slug 返回对应 Markdown 文档内容
// @Tags         docs
// @Produce      json
// @Param        slug  path  string  true  "文档标识（cli / mcp）"
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/docs/{slug} [get]
func GetDocContent(c *gin.Context) {
	slug := c.Param("slug")
	filename, ok := docSlugToFilename[slug]
	if !ok {
		response.Error(c, http.StatusNotFound, 404, "文档不存在")
		return
	}

	// 尝试从项目根目录的 docs/ 读取
	// 优先级：环境变量 QIM_DOCS_DIR > 相对路径 docs/ > 工作目录 docs/
	docsDir := os.Getenv("QIM_DOCS_DIR")
	if docsDir == "" {
		docsDir = "docs"
	}

	path := filepath.Join(docsDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		// 回退：尝试上一级目录
		path = filepath.Join("..", docsDir, filename)
		content, err = os.ReadFile(path)
		if err != nil {
			response.Error(c, http.StatusNotFound, 404, "文档文件未找到: "+filename)
			return
		}
	}

	// 只允许 .md 文件
	if !strings.HasSuffix(filename, ".md") {
		response.Error(c, http.StatusForbidden, 403, "不允许的文件类型")
		return
	}

	response.Success(c, gin.H{
		"slug":    slug,
		"title":   strings.TrimSuffix(filename, ".md"),
		"content": string(content),
	})
}

// ListDocs 列出所有可用文档。
// @Summary      列出可用文档
// @Description  返回所有可用文档的 slug 和标题
// @Tags         docs
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/v1/docs [get]
func ListDocs(c *gin.Context) {
	docs := make([]gin.H, 0, len(docSlugToFilename))
	for slug, filename := range docSlugToFilename {
		docs = append(docs, gin.H{
			"slug":  slug,
			"title": strings.TrimSuffix(filename, ".md"),
		})
	}
	response.Success(c, docs)
}
