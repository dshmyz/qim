package main

import (
	"github.com/spf13/cobra"
)

// newCompletionCmd 生成 shell 自动补全脚本。
// 用法: qim completion bash > /etc/bash_completion.d/qim
//
//	qim completion zsh > "${fpath[1]}/_qim"
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "生成 shell 自动补全脚本",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				die("不支持的 shell: %s（可选: bash|zsh|fish|powershell）", args[0])
			}
		},
	}
	return cmd
}
