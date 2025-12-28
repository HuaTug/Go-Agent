package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

type ChatOpenAI struct {
	Ctx          context.Context
	Model        string
	SystemPrompt string
	Tools        []mcp.Tool
	RagContext   string
	Message      []openai.ChatCompletionMessageParamUnion
	LLM          openai.Client
}

type LLMOption func(*ChatOpenAI)

func WithSystemPrompt(prompt string) LLMOption {
	return func(ai *ChatOpenAI) {
		ai.SystemPrompt = prompt
	}
}

func WithRagContext(ragPrompt string) LLMOption {
	return func(ai *ChatOpenAI) {
		ai.RagContext = ragPrompt
	}
}

func WithMessage(message []openai.ChatCompletionMessageParamUnion) LLMOption {
	return func(ai *ChatOpenAI) {
		ai.Message = message
	}
}

func WithLLMTools(tools []mcp.Tool) LLMOption {
	return func(ai *ChatOpenAI) {
		ai.Tools = tools
	}
}

func NewChatOpenAI(ctx context.Context, model string, opts ...LLMOption) *ChatOpenAI {
	if model == "" {
		panic("model is required")
	}
	var (
		apiKey  = os.Getenv(ChatGPTOpenAPIKEY)
		baseURL = os.Getenv(ChatGPTBaseURL)
	)
	// 使用默认的 DeepSeek API 配置
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	options := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}
	cli := openai.NewClient(options...)
	llm := &ChatOpenAI{
		Ctx:     ctx,
		Model:   model,
		LLM:     cli,
		Message: make([]openai.ChatCompletionMessageParamUnion, 0),
	}
	for _, opt := range opts {
		opt(llm)
	}
	if llm.SystemPrompt != "" {
		llm.Message = append(llm.Message, openai.SystemMessage(llm.SystemPrompt))
	}
	if llm.RagContext != "" {
		llm.Message = append(llm.Message, openai.UserMessage(llm.RagContext))
	}
	fmt.Println("init LLM successfully")
	return llm
}

func (c *ChatOpenAI) Chat(prompt string) (result string, toolCall []openai.ToolCallUnion) {
	fmt.Println("init chat...")
	if prompt != "" {
		c.Message = append(c.Message, openai.UserMessage(prompt))
	}
	toolsParam := MCPTool2OpenAITool(c.Tools)
	if len(toolsParam) == 0 {
		toolsParam = nil
	}
	fmt.Printf("Available tools count: %d\n", len(toolsParam))
	if len(toolsParam) > 0 {
		fmt.Println("First few tools:")
		for i, tool := range toolsParam {
			if i >= 3 {
				break
			}
			if tool.OfFunction != nil {
				fmt.Printf("  - %s\n", tool.OfFunction.Function.Name)
			}
		}
	}
	stream := c.LLM.Chat.Completions.NewStreaming(c.Ctx, openai.ChatCompletionNewParams{
		Messages:    c.Message,
		Seed:        openai.Int(0),
		Model:       c.Model,
		Tools:       toolsParam,
		Temperature: openai.Float(0.7),
		TopP:        openai.Float(0.6),
		MaxTokens:   openai.Int(12000),
	})
	acc := openai.ChatCompletionAccumulator{}
	var toolCalls []openai.ToolCallUnion
	result = ""
	finished := false
	fmt.Println("start chatting...")
	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)
		if content, ok := acc.JustFinishedContent(); ok {
			finished = true
			result = content
		}

		if tool, ok := acc.JustFinishedToolCall(); ok {
			fmt.Println("tool call finished:", tool.Index, tool.Name, tool.Arguments)
			toolCalls = append(toolCalls, openai.ToolCallUnion{
				ID: tool.ID,
				Function: openai.FunctionToolCallFunction{
					Name:      tool.Name,
					Arguments: tool.Arguments,
				},
			})
		}
		if refusal, ok := acc.JustFinishedRefusal(); ok {
			fmt.Println("refusal:", refusal)
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if !finished {
				result += delta
			}
		}
	}

	if len(acc.Choices) > 0 {
		c.Message = append(c.Message, acc.Choices[0].Message.ToParam())
		fmt.Printf("Response finish reason: %s\n", acc.Choices[0].FinishReason)
		fmt.Printf("Tool calls in response: %d\n", len(acc.Choices[0].Message.ToolCalls))

		// 如果流式处理中没有捕获到工具调用，从最终消息中提取
		if len(toolCalls) == 0 && len(acc.Choices[0].Message.ToolCalls) > 0 {
			fmt.Println("Extracting tool calls from final message...")
			for _, tc := range acc.Choices[0].Message.ToolCalls {
				toolCalls = append(toolCalls, openai.ToolCallUnion{
					ID: tc.ID,
					Function: openai.FunctionToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
	}

	if stream.Err() != nil {
		panic(stream.Err())
	}

	return result, toolCalls
}

func MCPTool2OpenAITool(mcpTools []mcp.Tool) []openai.ChatCompletionToolUnionParam {
	openAITools := make([]openai.ChatCompletionToolUnionParam, 0)
	for _, tool := range mcpTools {
		params := openai.FunctionParameters{
			"type":       tool.InputSchema.Type,
			"properties": tool.InputSchema.Properties,
			"required":   tool.InputSchema.Required,
		}
		// 关键兜底：若type为空，默认用object，避免OpenAI拒绝工具定义
		if t, ok := params["type"].(string); !ok || t == "" {
			params["type"] = "object"
		}
		openAITools = append(openAITools, openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        tool.Name,
					Description: openai.String(tool.Description),
					Parameters:  params,
				},
			},
		})
	}
	return openAITools
}
