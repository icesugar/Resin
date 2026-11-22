package publish

import (
	"encoding/json"
	"strings"
)

var supportedOutboundTypes = map[string]struct{}{
	"shadowsocks": {},
	"vmess":       {},
	"vless":       {},
	"trojan":      {},
	"hysteria2":   {},
	"hysteria":    {},
	"tuic":        {},
	"wireguard":   {},
	"http":        {},
	"socks":       {},
	"anytls":      {},
	"ssh":         {},
}

// IsSupported 判断 outbound 是否支持导出为 Clash 代理。
func IsSupported(raw []byte) bool {
	_, ok := supportedOutboundTypes[OutboundType(raw)]
	return ok
}

// OutboundType 返回 outbound 的 type 字段（小写）。
func OutboundType(raw []byte) string {
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(header.Type))
}

// Analyze 统计可导出与跳过的节点数量。
func Analyze(nodes []Node) Stats {
	stats := Stats{SkippedTypes: make(map[string]int)}
	for _, n := range nodes {
		stats.Matching++
		if !IsSupported(n.RawOptions) {
			stats.Skipped++
			typeName := OutboundType(n.RawOptions)
			if typeName == "" {
				typeName = "unknown"
			}
			stats.SkippedTypes[typeName]++
			continue
		}
		stats.Exportable++
	}
	return stats
}
