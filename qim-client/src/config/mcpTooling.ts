// qim-mcp 工具链单源：这些是技术标识（用户实际执行的命令），不随产品改名自动变化；
// 二进制名若调整，只改这里，前端所有提示与生成的配置保持一致。
export const MCP_CLI_BINARY = 'qim-mcp'

/** 生成 MCP stdio 接入配置 JSON（mcpServers 片段） */
export function buildMcpStdioConfig(serverUrl: string, token: string, key: string): string {
  return JSON.stringify({
    mcpServers: {
      [key]: {
        command: MCP_CLI_BINARY,
        args: ['--server', serverUrl, '--token', token],
      },
    },
  }, null, 2)
}
