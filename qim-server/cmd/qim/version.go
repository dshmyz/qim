package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本号",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("qim", version)
		},
	}
}
