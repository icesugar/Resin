package publish

import (
	"encoding/json"
	"fmt"
	"strings"
)

// clashProxyFromOutbound 把一个 sing-box outbound 转换为 Clash/Mihomo 代理配置。
func clashProxyFromOutbound(raw json.RawMessage, name string) (map[string]any, bool) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, false
	}

	typ := strings.ToLower(asString(m["type"]))
	server := asString(m["server"])
	port, ok := asInt(m["server_port"])
	if !ok || server == "" || port <= 0 {
		return nil, false
	}

	proxy := map[string]any{
		"name":   name,
		"server": server,
		"port":   port,
		"udp":    true,
	}

	switch typ {
	case "shadowsocks":
		if !buildShadowsocksProxy(proxy, m) {
			return nil, false
		}
	case "vmess":
		if !buildVMessProxy(proxy, m) {
			return nil, false
		}
	case "vless":
		if !buildVLESSProxy(proxy, m) {
			return nil, false
		}
	case "trojan":
		if !buildTrojanProxy(proxy, m) {
			return nil, false
		}
	case "hysteria2":
		if !buildHysteria2Proxy(proxy, m) {
			return nil, false
		}
	case "hysteria":
		if !buildHysteriaProxy(proxy, m) {
			return nil, false
		}
	case "tuic":
		if !buildTUICProxy(proxy, m) {
			return nil, false
		}
	case "wireguard":
		if !buildWireGuardProxy(proxy, m) {
			return nil, false
		}
	case "http":
		if !buildHTTPProxy(proxy, m) {
			return nil, false
		}
	case "socks":
		if !buildSocksProxy(proxy, m) {
			return nil, false
		}
	case "anytls":
		if !buildAnyTLSProxy(proxy, m) {
			return nil, false
		}
	case "ssh":
		if !buildSSHProxy(proxy, m) {
			return nil, false
		}
	default:
		return nil, false
	}
	return proxy, true
}

func buildShadowsocksProxy(proxy map[string]any, m map[string]any) bool {
	method := asString(m["method"])
	password := asString(m["password"])
	if method == "" || password == "" {
		return false
	}
	proxy["type"] = "ss"
	proxy["cipher"] = method
	proxy["password"] = password

	if plugin := asString(m["plugin"]); plugin != "" {
		applySSPlugin(proxy, plugin, asString(m["plugin_opts"]))
	}
	if uot := m["udp_over_tcp"]; uot != nil {
		proxy["udp-over-tcp"] = true
		if obj := asMap(uot); obj != nil {
			if version, ok := asInt(obj["version"]); ok && version > 0 {
				proxy["udp-over-tcp-version"] = version
			}
		}
	}
	applyDialFields(proxy, m)
	return true
}

func buildVMessProxy(proxy map[string]any, m map[string]any) bool {
	uuid := asString(m["uuid"])
	if uuid == "" {
		return false
	}
	proxy["type"] = "vmess"
	proxy["uuid"] = uuid
	alterID, _ := asInt(m["alter_id"])
	proxy["alterId"] = alterID
	cipher := firstString(m, "security", "cipher")
	if cipher == "" {
		cipher = "auto"
	}
	proxy["cipher"] = cipher
	if tlsEnabled(m) {
		proxy["tls"] = true
	}
	applyTLS(proxy, m, "servername")
	if !applyTransport(proxy, m) {
		return false
	}
	applyPacketEncoding(proxy, m)
	applyDialFields(proxy, m)
	return true
}

func buildVLESSProxy(proxy map[string]any, m map[string]any) bool {
	uuid := asString(m["uuid"])
	if uuid == "" {
		return false
	}
	proxy["type"] = "vless"
	proxy["uuid"] = uuid
	if flow := asString(m["flow"]); flow != "" {
		proxy["flow"] = flow
	}
	if tlsEnabled(m) {
		proxy["tls"] = true
	}
	applyTLS(proxy, m, "servername")
	if !applyTransport(proxy, m) {
		return false
	}
	applyPacketEncoding(proxy, m)
	applyDialFields(proxy, m)
	return true
}

func buildTrojanProxy(proxy map[string]any, m map[string]any) bool {
	password := asString(m["password"])
	if password == "" {
		return false
	}
	if tls := asMap(m["tls"]); tls != nil {
		if enabled, ok := asBool(tls["enabled"]); ok && !enabled {
			return false
		}
	}
	proxy["type"] = "trojan"
	proxy["password"] = password
	applyTLS(proxy, m, "sni")
	if !applyTransport(proxy, m) {
		return false
	}
	applyDialFields(proxy, m)
	return true
}

func buildHysteria2Proxy(proxy map[string]any, m map[string]any) bool {
	password := firstString(m, "password", "auth")
	if password == "" {
		return false
	}
	proxy["type"] = "hysteria2"
	proxy["password"] = password
	applyTLS(proxy, m, "sni")

	if obfs := asMap(m["obfs"]); obfs != nil {
		if typ := asString(obfs["type"]); typ != "" {
			proxy["obfs"] = typ
		}
		if pwd := asString(obfs["password"]); pwd != "" {
			proxy["obfs-password"] = pwd
		}
	}
	if ports := asStringSlice(m["server_ports"]); len(ports) > 0 {
		proxy["ports"] = strings.Join(ports, ",")
	}
	if up, ok := asInt(m["up_mbps"]); ok && up > 0 {
		proxy["up"] = fmt.Sprintf("%d Mbps", up)
	}
	if down, ok := asInt(m["down_mbps"]); ok && down > 0 {
		proxy["down"] = fmt.Sprintf("%d Mbps", down)
	}
	if hop := asString(m["hop_interval"]); hop != "" {
		proxy["hop-interval"] = hop
	}
	applyDialFields(proxy, m)
	return true
}

func buildHysteriaProxy(proxy map[string]any, m map[string]any) bool {
	proxy["type"] = "hysteria"
	if auth := firstString(m, "auth_str", "auth"); auth != "" {
		proxy["auth-str"] = auth
	}
	if up := asString(m["up"]); up != "" {
		proxy["up"] = up
	}
	if down := asString(m["down"]); down != "" {
		proxy["down"] = down
	}
	if obfs := asString(m["obfs"]); obfs != "" {
		proxy["obfs"] = obfs
	}
	if ports := asStringSlice(m["server_ports"]); len(ports) > 0 {
		proxy["ports"] = strings.Join(ports, ",")
	}
	if v, ok := asInt(m["recv_window_conn"]); ok {
		proxy["recv-window-conn"] = v
	}
	if v, ok := asInt(m["recv_window"]); ok {
		proxy["recv-window"] = v
	}
	if v, ok := asBool(m["disable_mtu_discovery"]); ok {
		proxy["disable-mtu-discovery"] = v
	}
	if hop := asString(m["hop_interval"]); hop != "" {
		proxy["hop-interval"] = hop
	}
	if strings.EqualFold(asString(m["network"]), "udp") {
		proxy["protocol"] = "udp"
	}
	applyTLS(proxy, m, "sni")
	applyDialFields(proxy, m)
	return true
}

func buildTUICProxy(proxy map[string]any, m map[string]any) bool {
	uuid := asString(m["uuid"])
	if uuid == "" {
		return false
	}
	proxy["type"] = "tuic"
	proxy["uuid"] = uuid
	if password := asString(m["password"]); password != "" {
		proxy["password"] = password
	}
	applyTLS(proxy, m, "sni")
	if v := asString(m["congestion_control"]); v != "" {
		proxy["congestion-controller"] = v
	}
	if v := asString(m["udp_relay_mode"]); v != "" {
		proxy["udp-relay-mode"] = v
	}
	if v, ok := asBool(m["zero_rtt_handshake"]); ok {
		proxy["reduce-rtt"] = v
	}
	if heartbeat := asString(m["heartbeat"]); heartbeat != "" {
		if ms, ok := durationToMillis(heartbeat); ok {
			proxy["heartbeat-interval"] = ms
		}
	}
	if v, ok := asBool(m["disable_sni"]); ok {
		proxy["disable-sni"] = v
	}
	applyDialFields(proxy, m)
	return true
}

func buildWireGuardProxy(proxy map[string]any, m map[string]any) bool {
	privateKey := asString(m["private_key"])
	if privateKey == "" {
		return false
	}

	publicKey := asString(m["peer_public_key"])
	server := asString(m["server"])
	port, _ := asInt(m["server_port"])
	preSharedKey := asString(m["pre_shared_key"])

	peers := getMapSlice(m["peers"])
	if len(peers) > 0 {
		peer := peers[0]
		if v := asString(peer["public_key"]); v != "" {
			publicKey = v
		}
		if v := asString(peer["server"]); v != "" {
			server = v
		}
		if v, ok := asInt(peer["server_port"]); ok {
			port = v
		}
		if v := asString(peer["pre_shared_key"]); v != "" {
			preSharedKey = v
		}
	}
	if publicKey == "" || server == "" || port <= 0 {
		return false
	}

	proxy["type"] = "wireguard"
	proxy["private-key"] = privateKey
	proxy["public-key"] = publicKey
	proxy["server"] = server
	proxy["port"] = port
	if preSharedKey != "" {
		proxy["pre-shared-key"] = preSharedKey
	}

	ipv4, ipv6 := splitWireGuardLocalAddress(asStringSlice(m["local_address"]))
	if ipv4 != "" {
		proxy["ip"] = ipv4
	}
	if ipv6 != "" {
		proxy["ipv6"] = ipv6
	}
	if reserved := asIntSlice(m["reserved"]); len(reserved) == 3 {
		proxy["reserved"] = reserved
	}
	if mtu, ok := asInt(m["mtu"]); ok && mtu > 0 {
		proxy["mtu"] = mtu
	}
	applyDialFields(proxy, m)
	return true
}

func buildHTTPProxy(proxy map[string]any, m map[string]any) bool {
	proxy["type"] = "http"
	if v := asString(m["username"]); v != "" {
		proxy["username"] = v
	}
	if v := asString(m["password"]); v != "" {
		proxy["password"] = v
	}
	if headers := asMap(m["headers"]); len(headers) > 0 {
		proxy["headers"] = headers
	}
	if tlsEnabled(m) {
		proxy["tls"] = true
	}
	applyTLS(proxy, m, "sni")
	applyDialFields(proxy, m)
	return true
}

func buildSocksProxy(proxy map[string]any, m map[string]any) bool {
	version := asString(m["version"])
	switch version {
	case "4", "4a":
		// Clash/Mihomo 不支持 SOCKS4/4a，直接跳过。
		return false
	case "", "5":
	default:
		return false
	}
	proxy["type"] = "socks5"
	if v := asString(m["username"]); v != "" {
		proxy["username"] = v
	}
	if v := asString(m["password"]); v != "" {
		proxy["password"] = v
	}
	if tlsEnabled(m) {
		proxy["tls"] = true
	}
	applyTLS(proxy, m, "sni")
	if strings.EqualFold(asString(m["network"]), "tcp") {
		proxy["udp"] = false
	}
	applyDialFields(proxy, m)
	return true
}

func buildAnyTLSProxy(proxy map[string]any, m map[string]any) bool {
	password := asString(m["password"])
	if password == "" {
		return false
	}
	proxy["type"] = "anytls"
	proxy["password"] = password
	applyTLS(proxy, m, "sni")
	if interval := asString(m["idle_session_check_interval"]); interval != "" {
		if sec, ok := durationToSeconds(interval); ok {
			proxy["idle-session-check-interval"] = sec
		}
	}
	if timeout := asString(m["idle_session_timeout"]); timeout != "" {
		if sec, ok := durationToSeconds(timeout); ok {
			proxy["idle-session-timeout"] = sec
		}
	}
	if v, ok := asInt(m["min_idle_session"]); ok {
		proxy["min-idle-session"] = v
	}
	applyDialFields(proxy, m)
	return true
}

func buildSSHProxy(proxy map[string]any, m map[string]any) bool {
	proxy["type"] = "ssh"
	if v := firstString(m, "user", "username"); v != "" {
		proxy["username"] = v
	}
	if v := asString(m["password"]); v != "" {
		proxy["password"] = v
	}
	if v := asString(m["private_key"]); v != "" {
		proxy["private-key"] = v
	}
	if v := asString(m["private_key_passphrase"]); v != "" {
		proxy["private-key-passphrase"] = v
	}
	if v := asStringSlice(m["host_key"]); len(v) > 0 {
		proxy["host-key"] = v
	}
	if v := asStringSlice(m["host_key_algorithms"]); len(v) > 0 {
		proxy["host-key-algorithms"] = v
	}
	if v := asString(m["client_version"]); v != "" {
		proxy["client-version"] = v
	}
	applyDialFields(proxy, m)
	return true
}
