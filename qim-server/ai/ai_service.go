package ai

import (
	"fmt"
	"log"
	"qim-server/config"
	"sync"
)

type AIService struct {
	config   *config.AIConfig
	factory  *ProviderFactory
	provider Provider
	mu       sync.RWMutex
}

func NewAIService(cfg *config.AIConfig) *AIService {
	svc := &AIService{
		config:  cfg,
		factory: NewProviderFactory(),
	}

	if err := svc.updateProvider(cfg); err != nil {
		log.Printf("[AI Service] Warning: Failed to initialize provider: %v", err)
	}

	return svc
}

func (s *AIService) UpdateConfig(cfg *config.AIConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg

	if err := s.updateProvider(cfg); err != nil {
		log.Printf("[AI Service] Failed to update provider: %v", err)
	}
}

func (s *AIService) updateProvider(cfg *config.AIConfig) error {
	provider, err := s.factory.CreateProvider(cfg)
	if err != nil {
		return err
	}
	s.provider = provider
	return nil
}

func (s *AIService) GetConfig() *config.AIConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *AIService) GetCompletion(messages []Message) (string, error) {
	s.mu.RLock()
	provider := s.provider
	s.mu.RUnlock()

	if provider == nil {
		return "", fmt.Errorf("AI provider not initialized")
	}

	filteredMessages := s.filterMessages(messages)
	return provider.Chat(filteredMessages)
}

// GetCompletionStream 流式获取AI完成
func (s *AIService) GetCompletionStream(messages []Message, onChunk func(chunk StreamChunk) error) error {
	s.mu.RLock()
	provider := s.provider
	s.mu.RUnlock()

	if provider == nil {
		return fmt.Errorf("AI provider not initialized")
	}

	filteredMessages := s.filterMessages(messages)
	return provider.ChatStream(filteredMessages, onChunk)
}

// 过滤消息内容，防止恶意输入
func (s *AIService) filterMessages(messages []Message) []Message {
	filtered := make([]Message, len(messages))
	for i, msg := range messages {
		filtered[i] = Message{
			Role:    msg.Role,
			Content: s.filterContent(msg.Content),
		}
	}
	return filtered
}

// 过滤内容，移除潜在的恶意内容
func (s *AIService) filterContent(content string) string {
	if len(content) > 10000 {
		content = content[:10000] + "..."
	}
	return content
}

func (s *AIService) IsConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.provider == nil {
		return false
	}

	return s.provider.IsConfigured()
}
