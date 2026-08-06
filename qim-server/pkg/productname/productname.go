// Package productname 提供产品名（品牌名）的单一来源。
//
// 默认 "QIM"。发布改名时通过构建期 ldflags 注入覆盖，无需改动源码：
//
//	go build -ldflags "-X github.com/dshmyz/qim/qim-server/pkg/productname.Name=青雀" ./main.go
//
// 与前端 __APP_PRODUCT_NAME__（admin/client 各自的 vite define）保持同一定义来源的分工：
// 前端负责 UI 展示名，本包负责后端发给 LLM 的提示词里的品牌名，避免提示词把内网改名前
// 的品牌名透给模型、被模型回显进回复。
//
// 注意：isProductQuestion 等「识别用户是否提到产品名」的触发词仍保留固定的 "QIM"/"qim"，
// 因为那是对用户输入的关键词匹配，不随展示名变化（见 smart_reply_graph.go）。
package productname

// Name 产品品牌名。生产默认 "QIM"；改名由 -ldflags "-X ...Name=新名" 覆盖。
var Name = "QIM"
