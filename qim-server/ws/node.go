package ws

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/utils"
)

func (h *Hub) startNodeCommunication() {
	// 这里可以实现节点发现和心跳检测
	logger.WithModule("WS").Info("节点间通信服务启动")
}

// broadcastToOtherNodes 通过 HTTP 向其他节点广播消息

func (h *Hub) broadcastToOtherNodes(message []byte) {
	for _, node := range h.nodes {
		if node == h.nodeID {
			continue // 跳过自身节点
		}

		// 构建其他节点的 URL
		nodeURL := h.nodeScheme + "://" + node + "/api/v1/node/broadcast"

		// 发送 HTTP 请求
		url := nodeURL
		utils.SafeGoWithLabel("node-broadcast", func() {
			resp, err := http.Post(url, "application/json", nil)
			if err != nil {
				logger.WithModule("WS").Error("向节点广播失败", "url", url, "error", err)
				return
			}
			defer resp.Body.Close()
		})
	}
}

func (h *Hub) sendToUserToOtherNodes(userID uint, message []byte) {
	for _, node := range h.nodes {
		if node == h.nodeID {
			continue // 跳过自身节点
		}

		// 构建其他节点的 URL
		nodeURL := h.nodeScheme + "://" + node + "/api/v1/node/send-to-user"

		// 构建请求体
		reqBody := map[string]interface{}{
			"user_id": userID,
			"message": string(message),
		}
		jsonBody, _ := json.Marshal(reqBody)

		// 发送 HTTP 请求
		url := nodeURL
		body := jsonBody
		utils.SafeGoWithLabel("node-send-user", func() {
			resp, err := http.Post(url, "application/json", bytes.NewReader(body))
			if err != nil {
				logger.WithModule("WS").Error("向节点发送用户消息失败", "url", url, "error", err)
				return
			}
			defer resp.Body.Close()
		})
	}
}
