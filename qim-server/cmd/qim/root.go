package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// outputFmt 控制输出格式："" (默认人类可读) 或 "json" (原始 JSON)。
var outputFmt string

// 产品名单源：产品改名只动这三个常量，所有用户可见文案经它们拼接。
// 协议标识（qusr_/qbot_ 令牌前缀、API 路径、表名）不属于产品名，另行全局替换。
const (
	// AppName 产品显示名（帮助标题、文档等场景）。
	AppName = "QIM"
	// CliName CLI 命令名（= 编译出的二进制名），命令示例与错误提示统一引用。
	CliName = "qim"
	// ConfigDirName 配置目录名（~/.qim）。
	ConfigDirName = ".qim"
)

// NewRootCmd 构建 cobra 根命令并注册所有子命令。
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   CliName,
		Short: fmt.Sprintf("%s agent CLI — 收发消息、管理任务/日历/笔记", AppName),
		Long: fmt.Sprintf(`%s - %s agent CLI（pull 底座）

纯 HTTP 客户端，不耦合 server 内部（service/handler），只依赖 REST 契约。
用 Bash 调用本 CLI，即可让 Claude Code/OpenCode 等 agent 在 %s 内收发消息、
管理待办、安排日历、读写笔记。

  %[4]s config set --server http://localhost:8080 --token qbot_...
  %[4]s messages list --thread <conv_id>
  %[4]s send --to <user_id> --thread <conv_id> --type markdown --content "hi"
  echo "增量" | %[4]s stream-stdin --to <user_id> --thread <conv_id>
  %[4]s note list / %[4]s note get <id> / %[4]s note create --title ...

默认以 bot 身份收发消息。需要以你本人身份操作时，先配置用户令牌
（%[4]s login，或客户端「设置 → 开发者设置」签发 qusr_ 长期令牌后
%[4]s config set --user-token qusr_...），再发消息：

  %[4]s send --as-user --to <用户名或用户ID> --content "hi"`, AppName, AppName, AppName, CliName),
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
		newAgentCmd(),
		newStreamCmd(),
		newStreamStdinCmd(),
		newTaskCmd(),
		newEventCmd(),
		newNotesCmd(),
		newTopSearchCmd(),
	)

	// cobra 自带 shell 补全生成（bash/zsh/fish/powershell）
	root.AddCommand(newCompletionCmd())

	return root
}
