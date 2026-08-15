package ws

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/utils"
)

// nodeHTTPClient 节点间通信 HTTP 客户端（带超时，避免发送方被对端拖死）。
var nodeHTTPClient = &http.Client{Timeout: 5 * time.Second}

func (h *Hub) startNodeCommunication() {
	// 这里可以实现节点发现和心跳检测
	logger.WithModule("WS").Info("节点间通信服务启动")
}

// nodeRequest 构建发往其他节点的 HTTP 请求（带 Node-Secret 认证头）。
func (h *Hub) nodeRequest(method, url string, body []byte) *http.Request {
	req, _ := http.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if h.nodeSecret != "" {
		req.Header.Set("Node-Secret", h.nodeSecret)
	}
	return req
}

// broadcastToOtherNodes 通过 HTTP 向其他节点广播消息。
// 对端 /api/v1/node/broadcast 由 NodeAuthMiddleware 保护：必须携带 Node-Secret 头；
// 且 body 为 {"message": ...}（对端 ShouldBindJSON，nil body 会 400）。
func (h *Hub) broadcastToOtherNodes(message []byte) {
	for _, node := range h.nodes {
		if node == h.nodeID {
			continue // 跳过自身节点
		}

		nodeURL := h.nodeScheme + "://" + node + "/api/v1/node/broadcast"
		body, _ := json.Marshal(map[string]string{"message": string(message)})

		utils.SafeGoWithLabel("node-broadcast", func() {
			resp, err := nodeHTTPClient.Do(h.nodeRequest(http.MethodPost, nodeURL, body))
			if err != nil {
				logger.WithModule("WS").Error("向节点广播失败", "url", nodeURL, "error", err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				logger.WithModule("WS").Warn("向节点广播返回非 200", "url", nodeURL, "status", resp.StatusCode)
			}
		})
	}
}

func (h *Hub) sendToUserToOtherNodes(userID uint, message []byte) {
	for _, node := range h.nodes {
		if node == h.nodeID {
			continue // 跳过自身节点
		}

		nodeURL := h.nodeScheme + "://" + node + "/api/v1/node/send-to-user"
		reqBody := map[string]interface{}{
			"user_id": userID,
			"message": string(message),
		}
		jsonBody, _ := json.Marshal(reqBody)

		utils.SafeGoWithLabel("node-send-user", func() {
			resp, err := nodeHTTPClient.Do(h.nodeRequest(http.MethodPost, nodeURL, jsonBody))
			if err != nil {
				logger.WithModule("WS").Error("向节点发送用户消息失败", "url", nodeURL, "error", err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				logger.WithModule("WS").Warn("向节点发送用户消息返回非 200", "url", nodeURL, "status", resp.StatusCode)
			}
		})
	}
}
