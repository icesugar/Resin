package publish

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// ClashOptions 控制 Clash 订阅输出。
type ClashOptions struct {
	GroupName    string
	ProviderOnly bool
}

type clashProxyGroup struct {
	Name      string   `yaml:"name"`
	Type      string   `yaml:"type"`
	Proxies   []string `yaml:"proxies"`
	URL       string   `yaml:"url,omitempty"`
	Interval  int      `yaml:"interval,omitempty"`
	Tolerance int      `yaml:"tolerance,omitempty"`
}

type clashConfig struct {
	MixedPort   int               `yaml:"mixed-port"`
	AllowLan    bool              `yaml:"allow-lan"`
	Mode        string            `yaml:"mode"`
	LogLevel    string            `yaml:"log-level"`
	Proxies     []map[string]any  `yaml:"proxies"`
	ProxyGroups []clashProxyGroup `yaml:"proxy-groups"`
	Rules       []string          `yaml:"rules"`
}

type clashProviderConfig struct {
	Proxies []map[string]any `yaml:"proxies"`
}

// RenderClash 把节点渲染为 Clash/Mihomo 订阅内容。
func RenderClash(nodes []Node, opts ClashOptions) (RenderResult, error) {
	names, groupName, autoGroupName := PrepareNodeNames(nodes, opts.GroupName)
	tagToName := BuildTagToNameMap(nodes, names)

	proxies := make([]map[string]any, 0, len(nodes))
	skippedTypes := map[string]int{}
	nodeNames := make([]string, 0, len(nodes))

	for i, n := range nodes {
		proxy, ok := clashProxyFromOutbound(n.RawOptions, names[i])
		if !ok {
			typeName := OutboundType(n.RawOptions)
			if typeName == "" {
				typeName = "unknown"
			}
			skippedTypes[typeName]++
			continue
		}
		remapClashDetour(proxy, tagToName, names[i])
		proxies = append(proxies, proxy)
		nodeNames = append(nodeNames, names[i])
	}

	sort.SliceStable(proxies, func(i, j int) bool {
		return asString(proxies[i]["name"]) < asString(proxies[j]["name"])
	})
	sort.Strings(nodeNames)

	skipped := 0
	for _, count := range skippedTypes {
		skipped += count
	}

	if opts.ProviderOnly {
		content, err := yaml.Marshal(clashProviderConfig{Proxies: proxies})
		if err != nil {
			return RenderResult{}, fmt.Errorf("marshal clash provider: %w", err)
		}
		return RenderResult{
			Content:      content,
			NodeCount:    len(proxies),
			Skipped:      skipped,
			SkippedTypes: skippedTypes,
			NodeNames:    nodeNames,
		}, nil
	}

	proxyNames := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		proxyNames = append(proxyNames, asString(proxy["name"]))
	}

	groups := make([]clashProxyGroup, 0, 2)
	mainGroupProxies := make([]string, 0, len(proxyNames)+2)
	if len(proxyNames) > 0 {
		mainGroupProxies = append(mainGroupProxies, autoGroupName, "DIRECT")
		groups = append(groups, clashProxyGroup{
			Name:      autoGroupName,
			Type:      "url-test",
			URL:       "https://www.gstatic.com/generate_204",
			Interval:  300,
			Tolerance: 50,
			Proxies:   proxyNames,
		})
	} else {
		mainGroupProxies = append(mainGroupProxies, "DIRECT")
	}
	mainGroupProxies = append(mainGroupProxies, proxyNames...)
	groups = append([]clashProxyGroup{{
		Name:    groupName,
		Type:    "select",
		Proxies: mainGroupProxies,
	}}, groups...)

	rules := append(clashDefaultDirectRules(), "MATCH,"+groupName)
	content, err := yaml.Marshal(clashConfig{
		MixedPort:   7890,
		AllowLan:    true,
		Mode:        "rule",
		LogLevel:    "info",
		Proxies:     proxies,
		ProxyGroups: groups,
		Rules:       rules,
	})
	if err != nil {
		return RenderResult{}, fmt.Errorf("marshal clash config: %w", err)
	}

	return RenderResult{
		Content:      content,
		NodeCount:    len(proxies),
		Skipped:      skipped,
		SkippedTypes: skippedTypes,
		NodeNames:    nodeNames,
	}, nil
}

func remapClashDetour(proxy map[string]any, tagToName map[string]string, self string) {
	detour, ok := proxy["dialer-proxy"].(string)
	if !ok || detour == "" {
		delete(proxy, "dialer-proxy")
		return
	}
	mapped, exists := tagToName[detour]
	if !exists || mapped == self {
		delete(proxy, "dialer-proxy")
		return
	}
	proxy["dialer-proxy"] = mapped
}
