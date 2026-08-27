package handler

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// broadcastChatJob 群发私聊任务状态。POST 立即返回 job_id，后端异步用固定 worker 池
// 并行发送，前端轮询 GET 获取真实成败——避免大用户量群发超过前端超时而"不知道发没发"。
// 所有字段（计数/状态）统一由 broadcastChatJobsMu 保护：worker 更新与查询快照走同一把锁，
// 避免原子写与非原子读混用的数据竞争。
type broadcastChatJob struct {
	ID      string `json:"job_id"`
	Status  string `json:"status"` // running | done
	Total   int    `json:"total"`
	Sent    int    `json:"sent"`
	Failed  int    `json:"failed"`
	Skipped int    `json:"skipped"`
	Content string `json:"-"`
	created time.Time
}

var (
	broadcastChatJobsMu sync.Mutex
	broadcastChatJobs   = make(map[string]*broadcastChatJob)
	broadcastChatJobSeq atomic.Uint64
)

const (
	maxBroadcastChatJobs   = 100
	maxBroadcastChatJobAge = 30 * time.Minute // 超过该时长仍 running 的任务视为卡死，逐出时强制回收
	broadcastChatWorkers   = 8
)

func nextBroadcastChatJobID() string {
	return fmt.Sprintf("bc-%d-%d", time.Now().UnixNano(), broadcastChatJobSeq.Add(1))
}

// newBroadcastChatJob 创建任务并放入内存存储。满时先逐出已完成的最早任务；
// 无已完成任务时，强制回收超时（>maxBroadcastChatJobAge）仍 running 的最早任务——
// 防止 DB 挂起/慢查询导致任务永不完成、map 无界增长砖住后续群发。
// 极新的 running 任务宁超容量也不逐出（否则前端轮询 404 而 worker 仍在投递）。
func newBroadcastChatJob(total int, content string) *broadcastChatJob {
	broadcastChatJobsMu.Lock()
	defer broadcastChatJobsMu.Unlock()
	for len(broadcastChatJobs) >= maxBroadcastChatJobs {
		var doneID string
		var doneOldest time.Time
		foundDone := false
		var staleID string
		var staleOldest time.Time
		foundStale := false
		for id, j := range broadcastChatJobs {
			if j.Status == "done" {
				if !foundDone || j.created.Before(doneOldest) {
					doneID, doneOldest, foundDone = id, j.created, true
				}
			} else if time.Since(j.created) > maxBroadcastChatJobAge {
				if !foundStale || j.created.Before(staleOldest) {
					staleID, staleOldest, foundStale = id, j.created, true
				}
			}
		}
		if foundDone {
			delete(broadcastChatJobs, doneID)
			continue
		}
		if foundStale {
			logger.WithModule("BroadcastChatMessage").Warn("回收超时仍 running 的群发任务", "job_id", staleID)
			delete(broadcastChatJobs, staleID)
			continue
		}
		break // 全是极新的 running 任务：不逐出活任务
	}
	job := &broadcastChatJob{
		ID:      nextBroadcastChatJobID(),
		Status:  "running",
		Total:   total,
		Content: content,
		created: time.Now(),
	}
	broadcastChatJobs[job.ID] = job
	return job
}

// snapshotBroadcastChatJob 在锁内拷贝任务，供响应序列化（避免与 worker 更新并发读竞争）。
func snapshotBroadcastChatJob(job *broadcastChatJob) *broadcastChatJob {
	broadcastChatJobsMu.Lock()
	defer broadcastChatJobsMu.Unlock()
	cp := *job
	return &cp
}

func getBroadcastChatJob(id string) *broadcastChatJob {
	broadcastChatJobsMu.Lock()
	defer broadcastChatJobsMu.Unlock()
	return broadcastChatJobs[id]
}

// runBroadcastChatJob 用固定 broadcastChatWorkers 个 worker goroutine 并行向目标用户群发私聊，
// 完成后置 done。固定池（channel 拉取）避免每用户一个 goroutine 在全员群发时创建数万协程。
// 每个目标独立 recover：任一发送 panic 只记失败，不拖垮服务进程。
func runBroadcastChatJob(job *broadcastChatJob, targetUsers []model.User, exclude map[uint]bool, systemUserID uint) {
	app := di.GlobalContainer
	msgSvc := app.MessageService
	convSvc := app.ConversationService

	inc := func(field *int) {
		broadcastChatJobsMu.Lock()
		*field++
		broadcastChatJobsMu.Unlock()
	}

	// 单个目标发送：recover 兜底，panic 视为失败
	sendOne := func(u model.User) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.WithModule("BroadcastChatMessage").Error("群发 worker panic", "user_id", u.ID, "err", r)
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		conv, convErr := convSvc.CreateSingleConversation(systemUserID, u.ID)
		if convErr != nil {
			return convErr
		}
		_, err = msgSvc.SendMessage(conv.ID, systemUserID, "text", job.Content, nil)
		return err
	}

	work := make(chan model.User)
	var wg sync.WaitGroup
	for i := 0; i < broadcastChatWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range work {
				if err := sendOne(u); err != nil {
					inc(&job.Failed)
					logger.WithModule("BroadcastChatMessage").Warn("发送私聊消息失败",
						"user_id", u.ID, "error", err)
				} else {
					inc(&job.Sent)
				}
			}
		}()
	}

	for _, u := range targetUsers {
		if exclude[u.ID] {
			inc(&job.Skipped)
			continue
		}
		work <- u
	}
	close(work)
	wg.Wait()

	// wg.Wait 建立 happens-before，此处读计数安全；置 done 持锁保证查询方可见
	broadcastChatJobsMu.Lock()
	job.Status = "done"
	broadcastChatJobsMu.Unlock()
	logger.WithModule("BroadcastChatMessage").Info("群发私聊完成",
		"job_id", job.ID, "total", job.Total, "sent", job.Sent, "failed", job.Failed, "skipped", job.Skipped)
}

// GetBroadcastChatJob 查询群发私聊任务进度/结果。
// GET /api/v1/system-messages/broadcast-chat/jobs/:id  (system_admin)
func GetBroadcastChatJob(c *gin.Context) {
	job := getBroadcastChatJob(c.Param("id"))
	if job == nil {
		response.NotFound(c, "任务不存在或已过期")
		return
	}
	response.Success(c, snapshotBroadcastChatJob(job))
}
