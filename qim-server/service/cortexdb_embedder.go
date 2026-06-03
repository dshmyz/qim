package service

import (
	"context"

	"github.com/dshmyz/qim/qim-server/ai"
)

const defaultEmbeddingDim = 1536

type CortexDBEmbedder struct {
	aiService *ai.AIService
	dim       int
}

func NewCortexDBEmbedder(aiService *ai.AIService) *CortexDBEmbedder {
	return &CortexDBEmbedder{
		aiService: aiService,
		dim:       defaultEmbeddingDim,
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

func (e *CortexDBEmbedder) Dim() int {
	return e.dim
}