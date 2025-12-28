package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== 高级Agent+LLM+RAG系统启动 ===")
	fmt.Println("1. 初始化基础系统...")

	// 1. 初始化基础agent系统
	allowDir, _ := os.Getwd()
	fmt.Println("工作目录:", allowDir)

	systemPrompt := `你是一个内容获取与文件写入助手。
	你必须且只能使用我提供的 MCP 工具完成任务：
	1) 使用 mcp-server-fetch 获取公开网页的内容（仅访问单页，遵守网站条款与 robots.txt，不进行批量或大规模抓取）。
	2) 对获取的公开内容进行简要总结。
	3) 使用 @modelcontextprotocol/server-filesystem 将结果写入当前目录的文件。
	不要自行访问网络或返回拒绝，直接调用上述工具完成。`

	fetchMcpCli := NewMCPClient(ctx, "uvx", nil, []string{"mcp-server-fetch"})
	fileMcpCli := NewMCPClient(ctx, "npx", nil, []string{"-y", "@modelcontextprotocol/server-filesystem", allowDir})

	agent := NewAgent(ctx, "deepseek-r1-0528", []*MCPClient{fetchMcpCli, fileMcpCli}, systemPrompt, "")

	fmt.Println("2. 初始化高级功能...")

	// 2. 初始化高级功能
	demo := NewDemoAdvancedFeatures(ctx)

	// 注册agent到分布式协调器
	err := demo.DistributedCoordinator.RegisterAgent("main-agent", agent)
	if err != nil {
		fmt.Printf("注册agent失败: %v\n", err)
	}

	fmt.Println("3. 运行演示...")

	// 3. 运行基础功能演示
	fmt.Println("\n=== 基础功能演示 ===")
	result := agent.Invoke("访问 https://news.ycombinator.com 首页公开内容，提取简要摘要，并将结果写入当前目录的 new.md（若存在则覆盖）。只使用提供的工具完成。")
	fmt.Println("基础任务结果:", result)

	// 4. 运行高级功能演示
	fmt.Println("\n=== 高级功能演示 ===")
	demo.RunAllDemos()

	// 5. 展示系统架构和技术亮点
	fmt.Println("\n=== 系统架构和技术亮点 ===")
	demo.ShowTechnicalHighlights()

	// 6. 关闭系统
	fmt.Println("\n=== 系统关闭 ===")
	agent.Close()
	demo.DistributedCoordinator.Stop()

	if demo.MonitoringSystem != nil {
		demo.MonitoringSystem.Shutdown()
	}

	fmt.Println("系统运行完成！")
}

// ShowTechnicalHighlights 展示技术亮点
func (daf *DemoAdvancedFeatures) ShowTechnicalHighlights() {
	highlights := []struct {
		Category string
		Features []string
	}{
		{
			Category: "分布式架构",
			Features: []string{
				"多agent任务协调与负载均衡",
				"任务队列和优先级调度",
				"容错和重试机制",
				"任务依赖关系管理",
			},
		},
		{
			Category: "高级RAG功能",
			Features: []string{
				"向量检索和语义搜索",
				"混合搜索（向量+关键词）",
				"多模态内容支持",
				"智能缓存和性能优化",
				"动态重排序机制",
			},
		},
		{
			Category: "监控和可观测性",
			Features: []string{
				"Prometheus指标收集",
				"OpenTelemetry链路追踪",
				"实时日志收集和分析",
				"智能告警系统",
				"系统健康监控",
			},
		},
		{
			Category: "性能优化",
			Features: []string{
				"多级缓存策略",
				"并发处理和资源管理",
				"流式响应和增量处理",
				"内存和CPU优化",
			},
		},
		{
			Category: "安全性和可靠性",
			Features: []string{
				"工具调用权限控制",
				"输入验证和清理",
				"错误处理和恢复机制",
				"资源限制和隔离",
			},
		},
	}

	fmt.Println("\n技术亮点总结:")
	for _, highlight := range highlights {
		fmt.Printf("\n%s:\n", highlight.Category)
		for _, feature := range highlight.Features {
			fmt.Printf("  ✓ %s\n", feature)
		}
	}

	// 展示面试讨论点
	fmt.Println("\n=== 面试技术讨论点 ===")
	discussionPoints := []string{
		"如何设计分布式agent系统的任务调度算法？",
		"向量检索在大规模数据下的性能优化策略",
		"RAG系统中如何处理多模态数据的统一表示？",
		"监控系统如何实现实时指标收集和告警触发？",
		"缓存策略在AI系统中的特殊考虑因素",
		"如何保证工具调用的安全性和可控性？",
		"流式处理在LLM应用中的实现难点",
		"系统容错和故障恢复的设计思路",
	}

	fmt.Println("可以深入讨论的技术问题:")
	for i, point := range discussionPoints {
		fmt.Printf("%d. %s\n", i+1, point)
	}
}
