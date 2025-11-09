package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/openai/openai-go/v3"
)

func TestChatOpenAI_Chat(t *testing.T) {
	ctx := context.Background()
	model := openai.ChatModelGPT3_5Turbo
	ai := NewChatOpenAI(ctx, model, WithRagContext(""), WithSystemPrompt(""))
	prompt := "hello!"
	result, err := ai.Chat(prompt)
	if err != nil {
		fmt.Println("err", err)
		return
	}
	fmt.Println("result", result)
}
