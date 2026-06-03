package handler

import (
	"context"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"sort"

	"github.com/gin-gonic/gin"
)

func AdminListVectorCollections(c *gin.Context) {
	vectorSvc := di.GlobalContainer.VectorService
	if vectorSvc == nil {
		response.InternalServerError(c, "向量服务不可用")
		return
	}

	db := vectorSvc.GetDB()
	if db == nil {
		response.InternalServerError(c, "向量数据库不可用")
		return
	}

	ctx := context.Background()
	collections, err := db.Vector().ListCollections(ctx)
	if err != nil {
		response.InternalServerError(c, "查询失败: "+err.Error())
		return
	}

	type CollectionInfo struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	infos := make([]CollectionInfo, 0, len(collections))
	for _, col := range collections {
		stats, err := db.Vector().GetCollectionStats(ctx, col.Name)
		count := 0
		if err == nil && stats != nil {
			count = int(stats.Count)
		}
		infos = append(infos, CollectionInfo{Name: col.Name, Count: count})
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	response.Success(c, infos)
}

func AdminGetVectorCollectionData(c *gin.Context) {
	collection := c.Param("name")
	if collection == "" {
		response.BadRequest(c, "集合名称不能为空")
		return
	}

	limit := 200
	vectorSvc := di.GlobalContainer.VectorService
	if vectorSvc == nil {
		response.InternalServerError(c, "向量服务不可用")
		return
	}

	ctx := context.Background()
	entries, err := vectorSvc.GetByCollection(ctx, collection, limit)
	if err != nil {
		response.InternalServerError(c, "查询失败: "+err.Error())
		return
	}

	type Entry struct {
		DocID    string            `json:"doc_id"`
		Content  string            `json:"content"`
		Metadata map[string]string `json:"metadata"`
	}
	items := make([]Entry, 0, len(entries))
	for _, e := range entries {
		docID := e.DocID
		if docID == "" {
			docID = e.ID
		}
		items = append(items, Entry{
			DocID:    docID,
			Content:  e.Content,
			Metadata: e.Metadata,
		})
	}

	response.Success(c, gin.H{
		"collection": collection,
		"entries":    items,
		"total":      len(items),
	})
}