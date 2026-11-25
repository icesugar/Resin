package publish

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

var clashReservedNames = map[string]struct{}{
	"DIRECT": {},
	"REJECT": {},
	"GLOBAL": {},
	"PASS":   {},
}

// SanitizeGroupName 清洗发布组名，并避免与 Clash 保留名冲突。
func SanitizeGroupName(raw string) string {
	name := sanitizeDisplayName(raw)
	if name == "" {
		name = "Resin"
	}
	// 规则行 MATCH,<group> 使用逗号分隔，组名中的逗号必须替换。
	name = strings.ReplaceAll(name, ",", "_")
	if isReservedClashName(name) {
		name += " Group"
	}
	return name
}

// SanitizeNodeName 清洗节点名，并避免与 Clash 保留名冲突。
func SanitizeNodeName(raw, fallback string) string {
	name := sanitizeDisplayName(raw)
	if name == "" {
		name = sanitizeDisplayName(fallback)
	}
	if name == "" {
		name = "node"
	}
	if isReservedClashName(name) {
		name += " Node"
	}
	return name
}

// SanitizeFilename 清洗用于 Content-Disposition 的文件名。
func SanitizeFilename(raw, fallback string) string {
	name := sanitizeDisplayName(raw)
	if name == "" {
		name = fallback
	}
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return r
	}, name)
	if strings.TrimSpace(name) == "" {
		return "subscription"
	}
	return name
}

// PrepareNodeNames 对节点排序并生成稳定、唯一的展示名。
// 返回的 autoGroupName 预留给 Clash url-test 组，避免节点名冲突。
func PrepareNodeNames(nodes []Node, groupName string) (names []string, safeGroupName string, autoGroupName string) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Name != nodes[j].Name {
			return nodes[i].Name < nodes[j].Name
		}
		return nodes[i].Hash < nodes[j].Hash
	})

	safeGroupName = SanitizeGroupName(groupName)
	autoGroupName = safeGroupName + "-Auto"
	reserved := map[string]struct{}{
		safeGroupName: {},
		autoGroupName: {},
	}
	baseNames := make([]string, len(nodes))
	for i, n := range nodes {
		baseNames[i] = nodeNameWithRegion(n.Region, SanitizeNodeName(n.Name, n.Hash))
	}
	return UniqueNames(baseNames, reserved), safeGroupName, autoGroupName
}

// nodeNameWithRegion 在节点名前添加出口国家代码，未知地区保持原名。
func nodeNameWithRegion(region, name string) string {
	code := strings.ToUpper(strings.TrimSpace(region))
	if len(code) != 2 {
		return name
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return name
		}
	}

	// 避免源节点名本身已以相同国家代码开头时重复添加。
	firstToken, _, _ := strings.Cut(name, " ")
	firstToken = strings.Trim(firstToken, "[](){}")
	if strings.EqualFold(firstToken, code) {
		return name
	}
	return sanitizeDisplayName(code + " " + name)
}

// UniqueNames 对名称做大小写不敏感的唯一化处理。
func UniqueNames(baseNames []string, reserved map[string]struct{}) []string {
	used := make(map[string]struct{}, len(baseNames)+len(reserved))
	for name := range reserved {
		used[strings.ToLower(name)] = struct{}{}
	}

	out := make([]string, len(baseNames))
	for i, base := range baseNames {
		name := base
		if _, exists := used[strings.ToLower(name)]; exists {
			for n := 2; ; n++ {
				candidate := fmt.Sprintf("%s #%d", base, n)
				if _, taken := used[strings.ToLower(candidate)]; !taken {
					name = candidate
					break
				}
			}
		}
		used[strings.ToLower(name)] = struct{}{}
		out[i] = name
	}
	return out
}

// BuildTagToNameMap 建立原始 sing-box tag 到导出名称的映射。
func BuildTagToNameMap(nodes []Node, names []string) map[string]string {
	out := make(map[string]string, len(nodes))
	for i, n := range nodes {
		if i >= len(names) {
			break
		}
		var header struct {
			Tag string `json:"tag"`
		}
		if err := json.Unmarshal(n.RawOptions, &header); err != nil {
			continue
		}
		tag := strings.TrimSpace(header.Tag)
		if tag == "" {
			continue
		}
		if _, exists := out[tag]; !exists {
			out[tag] = names[i]
		}
	}
	return out
}

// sanitizeDisplayName 清理换行、制表符与控制字符，并限制长度。
func sanitizeDisplayName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(raw))
	lastSpace := false
	for _, r := range raw {
		switch {
		case r == '\r' || r == '\n' || r == '\t':
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
		case r < 0x20 || r == 0x7f:
			continue
		default:
			b.WriteRune(r)
			lastSpace = r == ' '
		}
	}

	out := strings.TrimSpace(b.String())
	runes := []rune(out)
	if len(runes) > 120 {
		out = string(runes[:120])
	}
	return strings.TrimSpace(out)
}

// isReservedClashName 判断名称是否与 Clash 内置策略名冲突。
func isReservedClashName(name string) bool {
	_, ok := clashReservedNames[strings.ToUpper(strings.TrimSpace(name))]
	return ok
}
