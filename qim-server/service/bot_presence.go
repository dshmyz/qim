package service

import (
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
)

// botPresenceTTL 心跳存活窗口：超过该时长没有心跳即视为离线。
// agent loop 建议 30s 心跳一次，90s 窗口容忍两次丢包。
const botPresenceTTL = 90 * time.Second

// 外部 bot 在线状态（进程内注册表，不落库）：bot 定期 POST /bot/heartbeat，
// 消费方（bot 列表、运维视图）按 TTL 判定在线。语义是「该 bot 的驱动程序正在运行」，
// 而非 IM 意义上的在线——pull 模式下这是唯一可得的健康信号。
// 进程重启丢状态可自愈：loop 一个心跳周期内自动恢复在线。
var (
	botPresenceMu   sync.Mutex
	botPresenceLast = map[uint]time.Time{} // botID → 最近一次心跳
)

// MarkBotPresent 记录一次 bot 心跳。条目数以 bot 总数为上界，无需清扫。
func MarkBotPresent(botID uint) {
	botPresenceMu.Lock()
	botPresenceLast[botID] = time.Now()
	botPresenceMu.Unlock()
}

// BotOnline bot 是否在存活窗口内有过心跳。从未心跳过的 bot 一律离线
//（兼容未升级心跳的存量 agent，不误报在线）。
func BotOnline(botID uint) bool {
	botPresenceMu.Lock()
	defer botPresenceMu.Unlock()
	last, ok := botPresenceLast[botID]
	if !ok {
		return false
	}
	return time.Since(last) < botPresenceTTL
}

// FillBotOnline 为 bot 列表就地填充 Online 计算字段（model.Bot.Online 不落库）。
func FillBotOnline(bots []model.Bot) {
	for i := range bots {
		bots[i].Online = BotOnline(bots[i].ID)
	}
}
