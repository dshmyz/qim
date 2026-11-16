// qim-newclient 是 qim 的纯原生 mygo 桌面客户端。
// 具体启动逻辑在 internal/client，这里只保留可执行入口。
package main

import "qim-newclient/internal/client"

func main() { client.Run() }
