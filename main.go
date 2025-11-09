package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
)

func main() {
	ctx := context.Background()
	systemPrompt := "你是一个无敌的爬虫工具，你必须且只能使用我给你的这些工具进行作业"
	allowDir, _ := os.Getwd()
	fmt.Println("allowDir:", allowDir)
	fetchMcpCli := NewMCPClient(ctx, "uvx", nil, []string{"mcp-server-fetch"})
	fileMcpCli := NewMCPClient(ctx, "npx", nil, []string{"-y", "@modelcontextprotocol/server-filesystem", allowDir})
	agent := NewAgent(ctx, openai.ChatModelGPT3_5Turbo, []*MCPClient{fetchMcpCli, fileMcpCli}, systemPrompt, "")
	result := agent.Invoke("爬取https://news.ycombinator.com的内容，并且总结后保存到当前目录" + allowDir + "中的new.md的文件中")
	fmt.Println("result:", result)
}
