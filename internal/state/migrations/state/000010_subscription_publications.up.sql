-- 订阅发布：按过滤规则生成 Clash / sing-box 订阅内容。
CREATE TABLE IF NOT EXISTS subscription_publications (
	id                  TEXT PRIMARY KEY,
	name                TEXT NOT NULL UNIQUE,
	enabled             INTEGER NOT NULL DEFAULT 1,
	regex_filters_json  TEXT NOT NULL DEFAULT '[]',
	region_filters_json TEXT NOT NULL DEFAULT '[]',
	healthy_only        INTEGER NOT NULL DEFAULT 0,
	token               TEXT NOT NULL UNIQUE,
	created_at_ns       INTEGER NOT NULL,
	updated_at_ns       INTEGER NOT NULL
);
