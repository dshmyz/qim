package service

import (
	"crypto/md5"
	"encoding/binary"

	"github.com/dshmyz/qim/qim-server/model"
)

// FilterByRollout 根据客户端 ID 的哈希分桶判断是否命中灰度
// clientID 为空时仅放行 100% 全量版本（RolloutPercentage >= 100）
// clientID 非空时计算 MD5 哈希取前 2 字节 mod 100，与 RolloutPercentage 比较
func FilterByRollout(versions []model.ClientVersion, clientID string) []model.ClientVersion {
	var result []model.ClientVersion
	bucket := uint16(0)
	if clientID != "" {
		sum := md5.Sum([]byte(clientID))
		bucket = binary.BigEndian.Uint16(sum[:2]) % 100 // 0-99
	}
	for _, v := range versions {
		rollout := v.GetRolloutPercentage()
		if rollout >= 100 {
			result = append(result, v)
			continue
		}
		if clientID == "" {
			continue // 未携带 clientID 时跳过灰度版本
		}
		if bucket < uint16(rollout) {
			result = append(result, v)
		}
	}
	return result
}
