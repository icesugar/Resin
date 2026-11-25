package publish

const (
	chinaGeoIPRuleSetURL   = "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/cn.srs"
	chinaGeositeRuleSetURL = "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/cn.srs"
)

var directDomainSuffixes = []string{
	"baidu.com",
	"bdimg.com",
	"bdstatic.com",
	"bcebos.com",
	"qq.com",
	"qpic.cn",
	"gtimg.com",
	"wechat.com",
	"weixin.qq.com",
	"dingtalk.com",
	"dingtalkapps.com",
	"feishu.cn",
	"feishu.net",
	"larksuite.com",
	"alibaba.com",
	"alicdn.com",
	"alipay.com",
	"aliyun.com",
	"taobao.com",
	"tmall.com",
	"jd.com",
	"360buyimg.com",
	"weibo.com",
	"weibo.cn",
	"sina.com.cn",
	"sinaimg.cn",
	"bilibili.com",
	"hdslb.com",
	"douyin.com",
	"bytedance.com",
	"toutiao.com",
	"163.com",
	"126.net",
	"zhihu.com",
	"zhihuimg.com",
	"meituan.com",
	"dianping.com",
	"ctrip.com",
}

var directPrivateCIDRs = []string{
	"10.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"198.18.0.0/15",
}

var directPrivateCIDR6s = []string{
	"::1/128",
	"fc00::/7",
	"fe80::/10",
}

// clashDefaultDirectRules 返回 Clash/Mihomo 的内网、国内域名和国内 IP 直连规则。
func clashDefaultDirectRules() []string {
	rules := make([]string, 0, len(directPrivateCIDRs)+len(directPrivateCIDR6s)+len(directDomainSuffixes)+3)
	for _, cidr := range directPrivateCIDRs {
		rules = append(rules, "IP-CIDR,"+cidr+",DIRECT,no-resolve")
	}
	for _, cidr := range directPrivateCIDR6s {
		rules = append(rules, "IP-CIDR6,"+cidr+",DIRECT,no-resolve")
	}
	for _, suffix := range directDomainSuffixes {
		rules = append(rules, "DOMAIN-SUFFIX,"+suffix+",DIRECT")
	}
	return append(rules, "GEOSITE,CN,DIRECT", "GEOIP,CN,DIRECT")
}

// singboxDefaultDirectRules 返回 sing-box 的内网、国内域名和国内 IP 直连规则。
func singboxDefaultDirectRules() []singboxRouteRule {
	return []singboxRouteRule{
		{IPIsPrivate: true, Outbound: "direct"},
		{DomainSuffix: directDomainSuffixes, Outbound: "direct"},
		{RuleSet: []string{"geosite-cn"}, Outbound: "direct"},
		{RuleSet: []string{"geoip-cn"}, Outbound: "direct"},
	}
}

// singboxDefaultRuleSets 使用远程二进制规则集，兼容 sing-box 1.12+。
func singboxDefaultRuleSets() []singboxRuleSet {
	return []singboxRuleSet{
		{
			Type:           "remote",
			Tag:            "geosite-cn",
			Format:         "binary",
			URL:            chinaGeositeRuleSetURL,
			DownloadDetour: "direct",
		},
		{
			Type:           "remote",
			Tag:            "geoip-cn",
			Format:         "binary",
			URL:            chinaGeoIPRuleSetURL,
			DownloadDetour: "direct",
		},
	}
}
