package main

import (
	"github.com/spf13/cobra"
)

// outputFmt 控制输出格式："" (默认人类可读) 或 "json" (原始 JSON)。
var outputFmt string

// NewRootCmd 构建 cobra 根命令并注册所有子命令。
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "qim",
		Short: "QIM agent CLI — 收发消息、管理任务/日历",
		Long: `qim - QIM agent CLI（pull 底座）

纯 HTTP 客户端，不耦合 server 内部（service/handler），只依赖 REST 契约。
用 Bash 调用本 CLI，即可让 Claude Code/OpenCode 等 agent 在 QIM 内收发消息。

  qim config set --server http://localhost:8080 --token qbot_...
  qim messages list --thread <conv_id>
  qim send --to <user_id> --thread <conv_id> --type markdown --content "hi"
  echo "增量" | qim stream-stdin --to <user_id> --thread <conv_id>`,
		SilenceUsage: true, // 出错时不自动打印 usage（我们自己控制）
	}

	// 全局 persistent flag
	root.PersistentFlags().StringVar(&outputFmt, "output", "", "输出格式: json（原始 JSON）| id（裸 ID，方便脚本取值）| 空（人类可读）")

	// 注册子命令
	root.AddCommand(
		newConfigCmd(),
		newLoginCmd(),
		newVersionCmd(),
		newUpdateCmd(),
		newWhoamiCmd(),
		newConversationsCmd(),
		newMessagesCmd(),
		newSendCmd(),
		newGroupsCmd(),
		newStreamCmd(),
		newStreamStdinCmd(),
		newTaskCmd(),
		newEventCmd(),
	)

	// cobra 自带 shell 补全生成（bash/zsh/fish/powershell）
	root.AddCommand(newCompletionCmd())

	return root
}
