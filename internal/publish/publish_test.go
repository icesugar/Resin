package publish

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func testNodes() []Node {
	return []Node{
		{
			Hash:       "hash-hk",
			Name:       "HK 01",
			RawOptions: json.RawMessage(`{"type":"shadowsocks","tag":"ss-hk","server":"198.51.100.10","server_port":8388,"method":"aes-256-gcm","password":"secret"}`),
		},
		{
			Hash:       "hash-us",
			Name:       "US 01",
			RawOptions: json.RawMessage(`{"type":"vmess","tag":"vmess-us","server":"198.51.100.11","server_port":443,"uuid":"11111111-1111-1111-1111-111111111111","security":"auto","alter_id":0,"tls":{"enabled":true,"server_name":"example.com"},"transport":{"type":"ws","path":"/ws","headers":{"Host":"example.com"}}}`),
		},
		{
			Hash:       "hash-unsupported",
			Name:       "Direct",
			RawOptions: json.RawMessage(`{"type":"direct","tag":"direct"}`),
		},
	}
}

func TestGenerateToken(t *testing.T) {
	first, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	second, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if first == second {
		t.Fatal("tokens should differ")
	}
	raw, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if len(raw) != publicationTokenBytes {
		t.Fatalf("token bytes = %d, want %d", len(raw), publicationTokenBytes)
	}
}

func TestUniqueNamesCaseInsensitive(t *testing.T) {
	names := UniqueNames([]string{"A", "a", "A"}, nil)
	want := []string{"A", "a #2", "A #3"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestPrepareNodeNamesReservesGroups(t *testing.T) {
	nodes := []Node{
		{Hash: "h1", Name: "Group"},
		{Hash: "h2", Name: "Group-Auto"},
	}
	names, group, auto := PrepareNodeNames(nodes, "Group")
	if group != "Group" || auto != "Group-Auto" {
		t.Fatalf("group=%q auto=%q", group, auto)
	}
	if names[0] == group || names[0] == auto || names[1] == group || names[1] == auto {
		t.Fatalf("node names collide with groups: %v", names)
	}
	if names[0] == names[1] {
		t.Fatalf("node names not unique: %v", names)
	}
}

func TestAnalyzeSkipsUnsupported(t *testing.T) {
	stats := Analyze(testNodes())
	if stats.Matching != 3 || stats.Exportable != 2 || stats.Skipped != 1 {
		t.Fatalf("stats = %+v", stats)
	}
	if stats.SkippedTypes["direct"] != 1 {
		t.Fatalf("skipped types = %v", stats.SkippedTypes)
	}
}

func TestRenderClashFullConfig(t *testing.T) {
	result, err := RenderClash(testNodes(), ClashOptions{GroupName: "Resin Test"})
	if err != nil {
		t.Fatalf("RenderClash: %v", err)
	}
	if result.NodeCount != 2 || result.Skipped != 1 {
		t.Fatalf("result count=%d skipped=%d", result.NodeCount, result.Skipped)
	}

	var config struct {
		Proxies []map[string]any `yaml:"proxies"`
		Groups  []struct {
			Name    string   `yaml:"name"`
			Type    string   `yaml:"type"`
			Proxies []string `yaml:"proxies"`
		} `yaml:"proxy-groups"`
		Rules []string `yaml:"rules"`
	}
	if err := yaml.Unmarshal(result.Content, &config); err != nil {
		t.Fatalf("unmarshal clash config: %v\n%s", err, result.Content)
	}
	if len(config.Proxies) != 2 {
		t.Fatalf("proxies = %d, want 2\n%s", len(config.Proxies), result.Content)
	}
	if len(config.Groups) != 2 || config.Groups[0].Name != "Resin Test" || config.Groups[1].Type != "url-test" {
		t.Fatalf("unexpected groups: %+v", config.Groups)
	}
	if len(config.Rules) != 1 || config.Rules[0] != "MATCH,Resin Test" {
		t.Fatalf("rules = %v", config.Rules)
	}

	byType := map[string]map[string]any{}
	for _, proxy := range config.Proxies {
		byType[asString(proxy["type"])] = proxy
	}
	ss := byType["ss"]
	if ss == nil || asString(ss["cipher"]) != "aes-256-gcm" || asString(ss["password"]) != "secret" {
		t.Fatalf("shadowsocks proxy = %+v", ss)
	}
	vmess := byType["vmess"]
	if vmess == nil {
		t.Fatalf("missing vmess proxy: %+v", config.Proxies)
	}
	if vmess["tls"] != true || asString(vmess["servername"]) != "example.com" {
		t.Fatalf("vmess tls fields = %+v", vmess)
	}
	if asString(vmess["network"]) != "ws" {
		t.Fatalf("vmess network = %v", vmess["network"])
	}
	wsOpts, ok := vmess["ws-opts"].(map[string]any)
	if !ok || asString(wsOpts["path"]) != "/ws" {
		t.Fatalf("vmess ws-opts = %+v", vmess["ws-opts"])
	}
}

func TestRenderClashProviderOnly(t *testing.T) {
	result, err := RenderClash(testNodes(), ClashOptions{GroupName: "Resin Test", ProviderOnly: true})
	if err != nil {
		t.Fatalf("RenderClash provider: %v", err)
	}
	var config map[string]any
	if err := yaml.Unmarshal(result.Content, &config); err != nil {
		t.Fatalf("unmarshal provider: %v", err)
	}
	if _, ok := config["proxy-groups"]; ok {
		t.Fatalf("provider output should not contain proxy-groups: %s", result.Content)
	}
	proxies, ok := config["proxies"].([]any)
	if !ok || len(proxies) != 2 {
		t.Fatalf("provider proxies = %+v", config["proxies"])
	}
}

func TestRenderSingboxConfig(t *testing.T) {
	result, err := RenderSingbox(testNodes(), SingboxOptions{GroupName: "Resin Test"})
	if err != nil {
		t.Fatalf("RenderSingbox: %v", err)
	}
	if result.NodeCount != 2 || result.Skipped != 1 {
		t.Fatalf("result count=%d skipped=%d", result.NodeCount, result.Skipped)
	}

	var config struct {
		Outbounds []map[string]any `json:"outbounds"`
		Route     map[string]any   `json:"route"`
	}
	if err := json.Unmarshal(result.Content, &config); err != nil {
		t.Fatalf("unmarshal singbox: %v", err)
	}
	if len(config.Outbounds) != 4 {
		t.Fatalf("outbounds = %d, want 4: %s", len(config.Outbounds), result.Content)
	}
	selector := config.Outbounds[0]
	if asString(selector["type"]) != "selector" || asString(selector["tag"]) != "Resin Test" {
		t.Fatalf("selector = %+v", selector)
	}
	if asString(config.Route["final"]) != "Resin Test" {
		t.Fatalf("route final = %v", config.Route["final"])
	}
	// 原始 tag 必须被重写为导出名称，避免客户端出现不可读 tag。
	for _, outbound := range config.Outbounds[1:] {
		if asString(outbound["type"]) == "direct" {
			continue
		}
		if asString(outbound["tag"]) == "ss-hk" || asString(outbound["tag"]) == "vmess-us" {
			t.Fatalf("outbound tag was not rewritten: %+v", outbound)
		}
	}
}
