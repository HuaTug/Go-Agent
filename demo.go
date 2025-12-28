package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

// DemoAdvancedFeatures 演示高级功能
type DemoAdvancedFeatures struct {
	Ctx                    context.Context
	DistributedCoordinator *DistributedAgentCoordinator
	RAGEngine              *AdvancedRAGEngine
	MonitoringSystem       *MonitoringSystem
}

// NewDemoAdvancedFeatures 创建演示实例
func NewDemoAdvancedFeatures(ctx context.Context) *DemoAdvancedFeatures {
	// 初始化监控系统
	monitoring, err := NewMonitoringSystem(ctx, "advanced-agent-demo")
	if err != nil {
		log.Printf("Failed to initialize monitoring: %v", err)
		monitoring = nil
	}

	// 初始化分布式协调器
	coordinator := NewDistributedAgentCoordinator(ctx, 3)

	// 初始化RAG引擎（简化实现，使用内存存储）
	ragEngine := &AdvancedRAGEngine{
		Ctx:         ctx,
		VectorStore: &MemoryVectorStore{},
		Cache: &RAGCache{
			QueryCache:     make(map[string]*CacheEntry),
			EmbeddingCache: make(map[string][]float32),
			MaxSize:        100,
		},
	}

	return &DemoAdvancedFeatures{
		Ctx:                    ctx,
		DistributedCoordinator: coordinator,
		RAGEngine:              ragEngine,
		MonitoringSystem:       monitoring,
	}
}

// DemoDistributedTaskProcessing 演示分布式任务处理
func (daf *DemoAdvancedFeatures) DemoDistributedTaskProcessing() {
	fmt.Println("=== 分布式任务处理演示 ===")

	// 创建一些示例任务
	tasks := []*DistributedTask{
		{
			ID:       "task-1",
			Prompt:   "获取Hacker News首页内容并总结",
			Tools:    []string{"mcp-server-fetch", "filesystem"},
			Priority: 1,
			Timeout:  30 * time.Second,
		},
		{
			ID:           "task-2",
			Prompt:       "分析当前目录的文件结构并生成报告",
			Tools:        []string{"filesystem"},
			Priority:     2,
			Timeout:      60 * time.Second,
			Dependencies: []string{"task-1"},
		},
		{
			ID:           "task-3",
			Prompt:       "基于前两个任务的结果生成综合报告",
			Tools:        []string{"filesystem"},
			Priority:     3,
			Timeout:      45 * time.Second,
			Dependencies: []string{"task-1", "task-2"},
		},
	}

	// 提交任务
	for _, task := range tasks {
		taskID, err := daf.DistributedCoordinator.SubmitTask(task)
		if err != nil {
			fmt.Printf("提交任务失败: %v\n", err)
			continue
		}
		fmt.Printf("任务提交成功: %s\n", taskID)
	}

	// 监控任务状态
	daf.monitorTaskProgress()
}

// DemoAdvancedRAG 演示高级RAG功能
func (daf *DemoAdvancedFeatures) DemoAdvancedRAG() {
	fmt.Println("\n=== 高级RAG功能演示 ===")

	// 添加示例文档到RAG系统
	documents := []struct {
		content  string
		metadata map[string]interface{}
	}{
		{
			content: "MCP (Model Context Protocol) 是一个用于AI代理与外部工具通信的协议标准",
			metadata: map[string]interface{}{
				"source":   "mcp-docs",
				"category": "protocol",
				"version":  "1.0",
			},
		},
		{
			content: "RAG (Retrieval-Augmented Generation) 结合了检索和生成的能力，提升AI回答的准确性",
			metadata: map[string]interface{}{
				"source":   "ai-research",
				"category": "technique",
				"year":     2024,
			},
		},
		{
			content: "向量搜索使用嵌入向量来计算语义相似度，是现代搜索系统的重要组成部分",
			metadata: map[string]interface{}{
				"source":     "search-engine",
				"category":   "algorithm",
				"complexity": "advanced",
			},
		},
	}

	// 添加文档
	for i, doc := range documents {
		err := daf.RAGEngine.AddDocument(doc.content, doc.metadata, "text")
		if err != nil {
			fmt.Printf("添加文档失败 %d: %v\n", i+1, err)
		} else {
			fmt.Printf("文档 %d 添加成功\n", i+1)
		}
	}

	// 执行RAG搜索
	queries := []RAGQuery{
		{
			Query:      "什么是MCP协议？",
			TopK:       3,
			SearchType: "hybrid",
			Rerank:     true,
		},
		{
			Query:      "RAG技术的优势",
			TopK:       2,
			SearchType: "vector",
			Filters:    map[string]interface{}{"category": "technique"},
		},
	}

	for i, query := range queries {
		fmt.Printf("\n查询 %d: %s\n", i+1, query.Query)

		start := time.Now()
		results, err := daf.RAGEngine.Search(query)
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("搜索失败: %v\n", err)
			continue
		}

		fmt.Printf("搜索耗时: %v\n", duration)
		fmt.Printf("返回结果数: %d\n", len(results))

		for j, result := range results {
			fmt.Printf("  %d. 分数: %.3f, 类型: %s\n", j+1, result.Score, result.SearchType)
			fmt.Printf("     内容: %.50s...\n", result.Document.Content)
		}

		// 记录监控指标
		if daf.MonitoringSystem != nil {
			daf.MonitoringSystem.RecordRAGSearch(duration, len(results))
		}
	}
}

// DemoMonitoringAndObservability 演示监控和可观测性
func (daf *DemoAdvancedFeatures) DemoMonitoringAndObservability() {
	if daf.MonitoringSystem == nil {
		fmt.Println("\n=== 监控系统未初始化 ===")
		return
	}

	fmt.Println("\n=== 监控和可观测性演示 ===")

	// 记录一些示例指标
	daf.MonitoringSystem.RecordRequest("GET", 150*time.Millisecond, true)
	daf.MonitoringSystem.RecordLLMCall(2*time.Second, 500, true)
	daf.MonitoringSystem.RecordToolCall("filesystem", 100*time.Millisecond, true)
	daf.MonitoringSystem.RecordCacheHit()
	daf.MonitoringSystem.RecordCacheMiss()

	// 记录日志
	daf.MonitoringSystem.Log("info", "demo", "演示监控功能", map[string]interface{}{
		"component": "demo",
		"timestamp": time.Now().Unix(),
	}, nil)

	// 获取当前指标
	metrics := daf.MonitoringSystem.GetMetrics()
	fmt.Println("当前系统指标:")
	for name, value := range metrics {
		fmt.Printf("  %s: %v\n", name, value)
	}

	// 获取最新日志
	logs := daf.MonitoringSystem.GetLogs("", "", 5)
	fmt.Println("\n最新日志条目:")
	for i, log := range logs {
		fmt.Printf("  %d. [%s] %s: %s\n", i+1, log.Timestamp.Format("15:04:05"), log.Level, log.Message)
	}
}

// DemoPerformanceOptimization 演示性能优化功能
func (daf *DemoAdvancedFeatures) DemoPerformanceOptimization() {
	fmt.Println("\n=== 性能优化演示 ===")

	// 演示缓存效果
	fmt.Println("缓存性能测试:")

	query := RAGQuery{
		Query:      "测试缓存性能",
		TopK:       3,
		SearchType: "hybrid",
	}

	// 第一次查询（缓存未命中）
	start := time.Now()
	_, err := daf.RAGEngine.Search(query)
	firstDuration := time.Since(start)

	if err != nil {
		fmt.Printf("第一次查询失败: %v\n", err)
		return
	}

	// 第二次查询（缓存命中）
	start = time.Now()
	_, err = daf.RAGEngine.Search(query)
	secondDuration := time.Since(start)

	if err != nil {
		fmt.Printf("第二次查询失败: %v\n", err)
		return
	}

	fmt.Printf("第一次查询耗时: %v (缓存未命中)\n", firstDuration)
	fmt.Printf("第二次查询耗时: %v (缓存命中)\n", secondDuration)
	fmt.Printf("性能提升: %.1f%%\n",
		(float64(firstDuration)-float64(secondDuration))/float64(firstDuration)*100)

	// 演示并发处理
	daf.demoConcurrentProcessing()
}

// demoConcurrentProcessing 演示并发处理
func (daf *DemoAdvancedFeatures) demoConcurrentProcessing() {
	fmt.Println("\n并发处理测试:")

	queries := []RAGQuery{
		{Query: "并发查询1", TopK: 2},
		{Query: "并发查询2", TopK: 2},
		{Query: "并发查询3", TopK: 2},
		{Query: "并发查询4", TopK: 2},
	}

	start := time.Now()

	// 串行执行
	for i, query := range queries {
		_, err := daf.RAGEngine.Search(query)
		if err != nil {
			fmt.Printf("查询 %d 失败: %v\n", i+1, err)
		}
	}
	serialDuration := time.Since(start)

	fmt.Printf("串行执行总耗时: %v\n", serialDuration)
	fmt.Printf("平均每个查询耗时: %v\n", serialDuration/time.Duration(len(queries)))
}

// monitorTaskProgress 监控任务进度
func (daf *DemoAdvancedFeatures) monitorTaskProgress() {
	fmt.Println("\n监控任务进度:")

	// 模拟监控循环
	for i := 0; i < 5; i++ {
		time.Sleep(2 * time.Second)

		metrics := daf.DistributedCoordinator.GetMetrics()
		fmt.Printf("轮次 %d - 系统指标: %+v\n", i+1, metrics)

		if i == 2 && daf.MonitoringSystem != nil {
			// 模拟更新队列大小指标
			daf.MonitoringSystem.RecordMetricsUpdate("queue_size", metrics["queue_size"])
			daf.MonitoringSystem.RecordMetricsUpdate("active_workers", metrics["active_workers"])
		}
	}
}

// RunAllDemos 运行所有演示
func (daf *DemoAdvancedFeatures) RunAllDemos() {
	fmt.Println("开始运行高级功能演示...")

	daf.DemoDistributedTaskProcessing()
	daf.DemoAdvancedRAG()
	daf.DemoMonitoringAndObservability()
	daf.DemoPerformanceOptimization()

	fmt.Println("\n=== 所有演示完成 ===")
}

// MemoryVectorStore 内存向量存储（简化实现）
type MemoryVectorStore struct {
	documents []*Document
}

func (mvs *MemoryVectorStore) Store(doc *Document) error {
	mvs.documents = append(mvs.documents, doc)
	return nil
}

func (mvs *MemoryVectorStore) Search(query string, topK int) ([]*Document, error) {
	if len(mvs.documents) == 0 {
		return []*Document{}, nil
	}

	// 简化实现：返回所有文档
	if topK > len(mvs.documents) {
		topK = len(mvs.documents)
	}
	return mvs.documents[:topK], nil
}

func (mvs *MemoryVectorStore) Delete(docID string) error {
	for i, doc := range mvs.documents {
		if doc.ID == docID {
			mvs.documents = append(mvs.documents[:i], mvs.documents[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("document not found: %s", docID)
}

func (mvs *MemoryVectorStore) BatchStore(docs []*Document) error {
	mvs.documents = append(mvs.documents, docs...)
	return nil
}

// RecordMetricsUpdate 记录指标更新（辅助方法）
func (ms *MonitoringSystem) RecordMetricsUpdate(metricName string, value interface{}) {
	// 简化实现：记录日志
	ms.Log("info", "metrics", fmt.Sprintf("Metric update: %s = %v", metricName, value), nil, nil)
}
