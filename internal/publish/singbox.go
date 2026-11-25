package publish

import (
	"encoding/json"
	"fmt"
	"sort"
)

// SingboxOptions 控制 sing-box 订阅输出。
type SingboxOptions struct {
	GroupName string
}

type singboxInbound struct {
	Type       string `json:"type"`
	Tag        string `json:"tag"`
	Listen     string `json:"listen"`
	ListenPort int    `json:"listen_port"`
}

type singboxSelector struct {
	Type      string   `json:"type"`
	Tag       string   `json:"tag"`
	Outbounds []string `json:"outbounds"`
}

type singboxDirect struct {
	Type string `json:"type"`
	Tag  string `json:"tag"`
}

type singboxRouteRule struct {
	IPIsPrivate  bool     `json:"ip_is_private,omitempty"`
	DomainSuffix []string `json:"domain_suffix,omitempty"`
	RuleSet      []string `json:"rule_set,omitempty"`
	Outbound     string   `json:"outbound"`
}

type singboxRuleSet struct {
	Type           string `json:"type"`
	Tag            string `json:"tag"`
	Format         string `json:"format"`
	URL            string `json:"url"`
	DownloadDetour string `json:"download_detour,omitempty"`
}

type singboxRoute struct {
	Rules   []singboxRouteRule `json:"rules"`
	RuleSet []singboxRuleSet   `json:"rule_set,omitempty"`
	Final   string             `json:"final"`
}

type singboxConfig struct {
	Log       map[string]any   `json:"log"`
	Inbounds  []singboxInbound `json:"inbounds"`
	Outbounds []any            `json:"outbounds"`
	Route     singboxRoute     `json:"route"`
}

// RenderSingbox 把节点渲染为最小可用的 sing-box JSON 配置。
func RenderSingbox(nodes []Node, opts SingboxOptions) (RenderResult, error) {
	names, groupName, _ := PrepareNodeNames(nodes, opts.GroupName)
	tagToName := BuildTagToNameMap(nodes, names)

	outbounds := make([]any, 0, len(nodes)+2)
	nodeNames := make([]string, 0, len(nodes))
	skippedTypes := map[string]int{}

	for i, n := range nodes {
		if !IsSupported(n.RawOptions) {
			typeName := OutboundType(n.RawOptions)
			if typeName == "" {
				typeName = "unknown"
			}
			skippedTypes[typeName]++
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(n.RawOptions, &m); err != nil {
			skippedTypes["invalid"]++
			continue
		}
		m["tag"] = names[i]
		remapSingboxDetour(m, tagToName, names[i])
		outbounds = append(outbounds, m)
		nodeNames = append(nodeNames, names[i])
	}

	sort.Strings(nodeNames)

	selectorOutbounds := make([]string, 0, len(nodeNames)+1)
	selectorOutbounds = append(selectorOutbounds, "direct")
	selectorOutbounds = append(selectorOutbounds, nodeNames...)

	outbounds = append([]any{
		singboxSelector{
			Type:      "selector",
			Tag:       groupName,
			Outbounds: selectorOutbounds,
		},
	}, outbounds...)
	outbounds = append(outbounds, singboxDirect{Type: "direct", Tag: "direct"})

	config := singboxConfig{
		Log: map[string]any{"level": "info"},
		Inbounds: []singboxInbound{{
			Type:       "mixed",
			Tag:        "mixed-in",
			Listen:     "127.0.0.1",
			ListenPort: 2080,
		}},
		Outbounds: outbounds,
		Route: singboxRoute{
			Rules:   singboxDefaultDirectRules(),
			RuleSet: singboxDefaultRuleSets(),
			Final:   groupName,
		},
	}

	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return RenderResult{}, fmt.Errorf("marshal singbox config: %w", err)
	}

	skipped := 0
	for _, count := range skippedTypes {
		skipped += count
	}

	return RenderResult{
		Content:      content,
		NodeCount:    len(nodeNames),
		Skipped:      skipped,
		SkippedTypes: skippedTypes,
		NodeNames:    nodeNames,
	}, nil
}

func remapSingboxDetour(m map[string]any, tagToName map[string]string, self string) {
	detour := asString(m["detour"])
	if detour == "" {
		delete(m, "detour")
		return
	}
	mapped, exists := tagToName[detour]
	if !exists || mapped == self {
		delete(m, "detour")
		return
	}
	m["detour"] = mapped
}
