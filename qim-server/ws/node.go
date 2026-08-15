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
// 返回 error：http.NewRequest 对非法 method/畸形 URL 会失败（如 nodes 配置里带了
// 非法 scheme），此前忽略错误导致 nodeHTTPClient.Do(nil) 内部 panic，节点中继
// 整体崩溃。调用方须处理 err 并记录日志后返回，不能裸传 nil 请求。
func (h *Hub) nodeRequest(method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if h.nodeSecret != "" {
		req.Header.Set("Node-Secret", h.nodeSecret)
	}
	return req, nil
}

// broadcastToOtherNodes 通过 HTTP 向其他节点广播消息。
// 对端 /api/v1/node/broadcast 由 NodeAuthMiddleware 保护：必须携带 Node-Secret 头；
// 且 body 为 {"message": ..., "origin": ...}（对端 ShouldBindJSON，nil body 会 400）。
// origin 为发送方节点 ID：接收端据此识别来源并丢弃回环（DeliverBroadcastFromNode
// 遇 origin==本节点则不再投递），这是防跨节点消息风暴的关键。
func (h *Hub) broadcastToOtherNodes(message []byte) {
	for _, node := range h.nodes {
		if node == h.nodeID {
			continue // 跳过自身节点（nodeID 为 UUID，通常不匹配地址；真正去环靠接收端 origin 校验）
		}

		nodeURL := h.nodeScheme + "://" + node + "/api/v1/node/broadcast"
		body, _ := json.Marshal(map[string]interface{}{
			"message": string(message),
			"origin":  h.nodeID,
		})

		utils.SafeGoWithLabel("node-broadcast", func() {
			req, err := h.nodeRequest(http.MethodPost, nodeURL, body)
			if err != nil {
				logger.WithModule("WS").Error("构建节点广播请求失败", "url", nodeURL, "error", err)
				return
			}
			resp, err := nodeHTTPClient.Do(req)
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
			continue // 跳过自身节点（同上，去环靠接收端 origin 校验）
		}

		nodeURL := h.nodeScheme + "://" + node + "/api/v1/node/send-to-user"
		reqBody := map[string]interface{}{
			"user_id": userID,
			"message": string(message),
			"origin":  h.nodeID,
		}
		jsonBody, _ := json.Marshal(reqBody)

		utils.SafeGoWithLabel("node-send-user", func() {
			req, err := h.nodeRequest(http.MethodPost, nodeURL, jsonBody)
			if err != nil {
				logger.WithModule("WS").Error("构建节点用户消息请求失败", "url", nodeURL, "error", err)
				return
			}
			resp, err := nodeHTTPClient.Do(req)
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
