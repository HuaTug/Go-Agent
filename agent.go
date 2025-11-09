package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go/v3"
)

type Agent struct {
	MCPClient    []*MCPClient
	LLM          *ChatOpenAI
	Model        string
	SystemPrompt string
	Ctx          context.Context
	RAGCtx       string
}

func NewAgent(ctx context.Context, model string, mcpCli []*MCPClient, systemPrompt string, ragCtx string) *Agent {
	// 1. 激活所有的mcp client 拿到所有的tools
	tools := make([]mcp.Tool, 0)
	for _, item := range mcpCli {
		// 启动 stdio 传输
		err := item.Start()
		if err != nil {
			fmt.Println("mcp listen error:", err)
			continue
		}

		err = item.SetTools()
		if err != nil {
			fmt.Println("mcp set tools error:", err)
			continue
		}

		// 新增日志：打印该客户端的工具名称，便于确认工具是否正确注册
		for _, t := range item.GetTool() {
			fmt.Println("tool ready:", t.Name)
		}
		tools = append(tools, item.GetTool()...)
	}
	// 2. 激活并告诉llm有哪些tools
	llm := NewChatOpenAI(ctx, model, WithSystemPrompt(systemPrompt), WithLLMTools(tools), WithRagContext(ragCtx))
	fmt.Println("init LLM & Tools")
	return &Agent{
		MCPClient:    mcpCli,
		LLM:          llm,
		Model:        model,
		SystemPrompt: systemPrompt,
		RAGCtx:       ragCtx,
	}
}

func (a *Agent) Close() {
	for _, mcpClient := range a.MCPClient {
		mcpClient.Close()
	}
	fmt.Println("all close")
}

func (a *Agent) Invoke(prompt string) string {
	if a.LLM == nil {
		return ""
	}
	response, toolCalls := a.LLM.Chat(prompt)
	fmt.Println("toolCalls", toolCalls)
	for len(toolCalls) > 0 {
		fmt.Println("response", response)
		for _, toolCall := range toolCalls {
			for _, mcpClient := range a.MCPClient {
				for _, mcpTool := range mcpClient.GetTool() {
					if mcpTool.Name == toolCall.Function.Name {
						fmt.Println("tool use", toolCall.ID, toolCall.Function.Name, toolCall.Function.Arguments)
						toolText, err := mcpClient.CallTool(toolCall.Function.Name, toolCall.Function.Arguments)
						if err != nil {
							fmt.Println("call tool error:", err)
							continue
						}
						a.LLM.Message = append(a.LLM.Message, openai.ToolMessage(toolText, toolCall.ID))
					}
				}
			}
		}
		// 二次对话（空 prompt 也会发起请求）
		response, toolCalls = a.LLM.Chat("")
	}
	a.Close()
	return response
}
