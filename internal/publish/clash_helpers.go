package publish

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// applyTLS 把 sing-box TLS 字段映射为 Clash 字段。
// serverNameKey 在 VMess/VLESS 中为 servername，其余协议通常为 sni。
func applyTLS(proxy map[string]any, m map[string]any, serverNameKey string) {
	tls := asMap(m["tls"])
	if tls == nil {
		return
	}
	if enabled, ok := asBool(tls["enabled"]); ok && !enabled {
		return
	}
	if serverName := asString(tls["server_name"]); serverName != "" {
		proxy[serverNameKey] = serverName
	}
	if insecure, ok := asBool(tls["insecure"]); ok && insecure {
		proxy["skip-cert-verify"] = true
	}
	if disableSNI, ok := asBool(tls["disable_sni"]); ok && disableSNI {
		proxy["disable-sni"] = true
	}
	if alpn := asStringSlice(tls["alpn"]); len(alpn) > 0 {
		proxy["alpn"] = alpn
	}
	if utls := asMap(tls["utls"]); utls != nil {
		if fingerprint := asString(utls["fingerprint"]); fingerprint != "" {
			proxy["client-fingerprint"] = fingerprint
		}
	}
	if reality := asMap(tls["reality"]); reality != nil {
		publicKey := asString(reality["public_key"])
		shortID := asString(reality["short_id"])
		if publicKey != "" || shortID != "" {
			opts := map[string]any{}
			if publicKey != "" {
				opts["public-key"] = publicKey
			}
			if shortID != "" {
				opts["short-id"] = shortID
			}
			proxy["reality-opts"] = opts
		}
	}
	if caPath := asString(tls["certificate_path"]); caPath != "" {
		proxy["ca"] = caPath
	}
	if certs := asStringSlice(tls["certificate"]); len(certs) > 0 {
		proxy["ca-str"] = certs[0]
	}
}

// applyTransport 把 sing-box transport 映射为 Clash network 配置。
// 返回 false 表示该 transport 无法用 Clash 表达。
func applyTransport(proxy map[string]any, m map[string]any) bool {
	transport := asMap(m["transport"])
	if transport == nil {
		return true
	}

	switch strings.ToLower(asString(transport["type"])) {
	case "", "tcp":
		return true
	case "ws":
		proxy["network"] = "ws"
		opts := map[string]any{}
		if path := asString(transport["path"]); path != "" {
			opts["path"] = path
		}
		if headers := asMap(transport["headers"]); len(headers) > 0 {
			opts["headers"] = headers
		}
		if maxEarly, ok := asInt(transport["max_early_data"]); ok && maxEarly > 0 {
			opts["max-early-data"] = maxEarly
		}
		if headerName := asString(transport["early_data_header_name"]); headerName != "" {
			opts["early-data-header-name"] = headerName
		}
		if len(opts) > 0 {
			proxy["ws-opts"] = opts
		}
		return true
	case "grpc":
		proxy["network"] = "grpc"
		if serviceName := asString(transport["service_name"]); serviceName != "" {
			proxy["grpc-opts"] = map[string]any{"grpc-service-name": serviceName}
		}
		return true
	case "http":
		proxy["network"] = "h2"
		opts := map[string]any{}
		if hosts := asStringSlice(transport["host"]); len(hosts) > 0 {
			opts["host"] = hosts
		}
		if path := asString(transport["path"]); path != "" {
			opts["path"] = path
		}
		if len(opts) > 0 {
			proxy["h2-opts"] = opts
		}
		return true
	case "httpupgrade":
		proxy["network"] = "httpupgrade"
		opts := map[string]any{}
		if path := asString(transport["path"]); path != "" {
			opts["path"] = path
		}
		if hosts := asStringSlice(transport["host"]); len(hosts) > 0 {
			opts["host"] = hosts[0]
		}
		if headers := asMap(transport["headers"]); len(headers) > 0 {
			opts["headers"] = headers
		}
		if len(opts) > 0 {
			proxy["http-upgrade-opts"] = opts
		}
		return true
	default:
		// quic / kcp 等 Clash 兼容性差的 transport 直接跳过该节点。
		return false
	}
}

// applyDialFields 把 sing-box 拨号字段映射为 Clash 字段。
func applyDialFields(proxy map[string]any, m map[string]any) {
	if v := asString(m["bind_interface"]); v != "" {
		proxy["interface-name"] = v
	}
	if v, ok := asInt(m["routing_mark"]); ok {
		proxy["routing-mark"] = v
	} else if v := asString(m["routing_mark"]); v != "" {
		proxy["routing-mark"] = v
	}
	if v, ok := asBool(m["tcp_fast_open"]); ok {
		proxy["tfo"] = v
	}
	if v, ok := asBool(m["tcp_multi_path"]); ok {
		proxy["mptcp"] = v
	}
	if v, ok := asBool(m["udp_fragment"]); ok {
		proxy["udp-fragment"] = v
	}
	if strategy := asString(m["domain_strategy"]); strategy != "" {
		if ipVersion := domainStrategyToIPVersion(strategy); ipVersion != "" {
			proxy["ip-version"] = ipVersion
		}
	}
	if detour := asString(m["detour"]); detour != "" {
		proxy["dialer-proxy"] = detour
	}
	if smux := asMap(m["multiplex"]); smux != nil {
		applyMultiplex(proxy, smux)
	}
}

func applyMultiplex(proxy map[string]any, smux map[string]any) {
	enabled, ok := asBool(smux["enabled"])
	if !ok || !enabled {
		return
	}
	opts := map[string]any{"enabled": true}
	if v := asString(smux["protocol"]); v != "" {
		opts["protocol"] = v
	}
	if v, ok := asInt(smux["max_connections"]); ok {
		opts["max-connections"] = v
	}
	if v, ok := asInt(smux["min_streams"]); ok {
		opts["min-streams"] = v
	}
	if v, ok := asBool(smux["padding"]); ok {
		opts["padding"] = v
	}
	proxy["smux"] = opts
}

func applyPacketEncoding(proxy map[string]any, m map[string]any) {
	if v := asString(m["packet_encoding"]); v != "" {
		proxy["packet-encoding"] = v
	}
}

// applySSPlugin 把 sing-box shadowsocks plugin 映射为 Clash plugin / plugin-opts。
func applySSPlugin(proxy map[string]any, plugin, pluginOpts string) {
	name := strings.ToLower(strings.TrimSpace(plugin))
	opts := parsePluginOptions(pluginOpts)

	switch name {
	case "obfs-local", "obfs", "simple-obfs":
		proxy["plugin"] = "obfs"
		out := map[string]any{}
		if v := firstPluginOption(opts, "obfs", "mode"); v != "" {
			out["mode"] = v
		}
		if v := firstPluginOption(opts, "obfs-host", "host"); v != "" {
			out["host"] = v
		}
		if len(out) > 0 {
			proxy["plugin-opts"] = out
		}
	case "v2ray-plugin":
		proxy["plugin"] = "v2ray-plugin"
		out := map[string]any{}
		if v := firstPluginOption(opts, "mode"); v != "" {
			out["mode"] = v
		}
		if v := firstPluginOption(opts, "host"); v != "" {
			out["host"] = v
		}
		if v := firstPluginOption(opts, "path"); v != "" {
			out["path"] = v
		}
		if v, ok := pluginOptionBool(opts, "tls"); ok && v {
			out["tls"] = true
		}
		if v, ok := pluginOptionBool(opts, "mux"); ok && v {
			out["mux"] = true
		}
		if len(out) > 0 {
			proxy["plugin-opts"] = out
		}
	case "shadow-tls":
		proxy["plugin"] = "shadow-tls"
		out := map[string]any{}
		if v := firstPluginOption(opts, "host"); v != "" {
			out["host"] = v
		}
		if v := firstPluginOption(opts, "password"); v != "" {
			out["password"] = v
		}
		if v := firstPluginOption(opts, "version"); v != "" {
			out["version"] = v
		}
		if len(out) > 0 {
			proxy["plugin-opts"] = out
		}
	case "restls":
		proxy["plugin"] = "restls"
		out := map[string]any{}
		if v := firstPluginOption(opts, "host"); v != "" {
			out["host"] = v
		}
		if v := firstPluginOption(opts, "password"); v != "" {
			out["password"] = v
		}
		if v := firstPluginOption(opts, "version-hint"); v != "" {
			out["version-hint"] = v
		}
		if len(out) > 0 {
			proxy["plugin-opts"] = out
		}
	default:
		if name != "" {
			proxy["plugin"] = name
		}
		if len(opts) > 0 {
			proxy["plugin-opts"] = opts
		}
	}
}

// parsePluginOptions 解析 "k=v;k2=v2;flag" 形式的 plugin_options。
func parsePluginOptions(raw string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, hasValue := strings.Cut(part, "=")
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if !hasValue {
			out[strings.ToLower(key)] = "true"
			continue
		}
		out[strings.ToLower(key)] = strings.TrimSpace(value)
	}
	return out
}

func firstPluginOption(opts map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(opts[strings.ToLower(key)]); v != "" {
			return v
		}
	}
	return ""
}

func pluginOptionBool(opts map[string]string, key string) (bool, bool) {
	v, ok := opts[strings.ToLower(key)]
	if !ok {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on", "":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

func tlsEnabled(m map[string]any) bool {
	tls := asMap(m["tls"])
	if tls == nil {
		return false
	}
	if enabled, ok := asBool(tls["enabled"]); ok {
		return enabled
	}
	return true
}

func domainStrategyToIPVersion(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "prefer_ipv4":
		return "prefer-ipv4"
	case "prefer_ipv6":
		return "prefer-ipv6"
	case "ipv4_only":
		return "ipv4"
	case "ipv6_only":
		return "ipv6"
	default:
		return ""
	}
}

func durationToMillis(raw string) (int, bool) {
	d, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, false
	}
	return int(d / time.Millisecond), true
}

func durationToSeconds(raw string) (int, bool) {
	d, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, false
	}
	return int(d / time.Second), true
}

func splitWireGuardLocalAddress(addresses []string) (ipv4 string, ipv6 string) {
	for _, addr := range addresses {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		host := addr
		if idx := strings.IndexByte(host, '/'); idx >= 0 {
			host = host[:idx]
		}
		if strings.Contains(host, ":") {
			if ipv6 == "" {
				ipv6 = host
			}
			continue
		}
		if ipv4 == "" {
			ipv4 = host
		}
	}
	return ipv4, ipv6
}

func getMapSlice(v any) []map[string]any {
	switch t := v.(type) {
	case []map[string]any:
		return t
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, item := range t {
			if m := asMap(item); m != nil {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func asString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func asInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case uint64:
		return int(t), true
	case float64:
		if math.Trunc(t) == t {
			return int(t), true
		}
		return 0, false
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func asBool(v any) (bool, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
	case float64:
		switch t {
		case 1:
			return true, true
		case 0:
			return false, true
		}
	}
	return false, false
}

func asStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return append([]string(nil), t...)
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s := asString(item); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if s := strings.TrimSpace(t); s != "" {
			return []string{s}
		}
	}
	return nil
}

func asIntSlice(v any) []int {
	switch t := v.(type) {
	case []int:
		return append([]int(nil), t...)
	case []any:
		out := make([]int, 0, len(t))
		for _, item := range t {
			if n, ok := asInt(item); ok {
				out = append(out, n)
			}
		}
		return out
	}
	return nil
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v := asString(m[key]); v != "" {
			return v
		}
	}
	return ""
}
