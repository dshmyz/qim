package service

import (
	"context"
	"sync"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

const defaultEmbeddingDim = 1536

type CortexDBEmbedder struct {
	aiService *ai.AIService
	dim       int
	once      sync.Once
}

// NewCortexDBEmbedder 创建 embedder，维度在首次 Dim() 调用时自动探测。
func NewCortexDBEmbedder(aiService *ai.AIService) *CortexDBEmbedder {
	return &CortexDBEmbedder{
		aiService: aiService,
		// dim=0 表示未设置，Dim() 时自动探测
	}
}

// NewCortexDBEmbedderWithDim 创建 embedder 并显式指定维度（跳过自动探测）。
func NewCortexDBEmbedderWithDim(aiService *ai.AIService, dim int) *CortexDBEmbedder {
	return &CortexDBEmbedder{
		aiService: aiService,
		dim:       dim,
	}
}

func (e *CortexDBEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return e.aiService.Embed(text)
}

func (e *CortexDBEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.aiService.Embed(text)
		if err != nil {
			return nil, err
		}
		results[i] = vec
	}
	return results, nil
}

// Dim 返回嵌入维度。首次调用时通过一次探测性 embedding 自动探测；
// 如果已通过 NewCortexDBEmbedderWithDim 显式指定则直接返回。
// 探测失败时回退到 defaultEmbeddingDim (1536)。
func (e *CortexDBEmbedder) Dim() int {
	e.once.Do(func() {
		if e.dim > 0 {
			return // 已显式指定
		}
		vec, err := e.aiService.Embed("dimension probe")
		if err != nil {
			logger.WithModule("CortexDB").Warn("embedding 维度自动探测失败，回退到默认值", "default", defaultEmbeddingDim, "err", err)
			e.dim = defaultEmbeddingDim
			return
		}
		e.dim = len(vec)
		logger.WithModule("CortexDB").Info("自动探测 embedding 维度", "dim", e.dim)
	})
	return e.dim
}
