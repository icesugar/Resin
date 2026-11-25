// Package publish 负责把节点池中的 sing-box outbound 渲染为客户端订阅内容。
package publish

import "encoding/json"

// Node 表示一个待导出的节点。
type Node struct {
	Hash       string
	Name       string
	Region     string
	RawOptions json.RawMessage
}

// Stats 表示一次节点分析的统计结果。
type Stats struct {
	Matching     int
	Exportable   int
	Skipped      int
	SkippedTypes map[string]int
}

// RenderResult 表示渲染后的订阅内容与统计信息。
type RenderResult struct {
	Content      []byte
	NodeCount    int
	Skipped      int
	SkippedTypes map[string]int
	NodeNames    []string
}
