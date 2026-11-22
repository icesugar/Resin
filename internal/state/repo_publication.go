package state

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Resinat/Resin/internal/model"
	"github.com/Resinat/Resin/internal/platform"
)

// rowScanner 抽象 sql.Row / sql.Rows 的 Scan 方法，便于复用扫描逻辑。
type rowScanner interface {
	Scan(dest ...any) error
}

// UpsertSubscriptionPublication 插入或更新一条订阅发布配置。
// 名称冲突或 token 冲突会返回 ErrConflict。
func (r *StateRepo) UpsertSubscriptionPublication(p model.SubscriptionPublication) error {
	p.Name = platform.NormalizePlatformName(p.Name)
	if err := platform.ValidatePlatformName(p.Name); err != nil {
		return fmt.Errorf("subscription publication name: %w", err)
	}
	if _, err := platform.CompileRegexFilters(p.RegexFilters); err != nil {
		return fmt.Errorf("subscription publication regex_filters: %w", err)
	}
	if err := platform.ValidateRegionFilters(p.RegionFilters); err != nil {
		return fmt.Errorf("subscription publication region_filters: %w", err)
	}
	if strings.TrimSpace(p.Token) == "" {
		return fmt.Errorf("subscription publication token: must be non-empty")
	}

	regexJSON, err := encodeStringSliceJSON(p.RegexFilters)
	if err != nil {
		return fmt.Errorf("encode subscription publication %s regex_filters: %w", p.ID, err)
	}
	regionJSON, err := encodeStringSliceJSON(p.RegionFilters)
	if err != nil {
		return fmt.Errorf("encode subscription publication %s region_filters: %w", p.ID, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	_, err = r.db.Exec(`
		INSERT INTO subscription_publications (
			id, name, enabled, regex_filters_json, region_filters_json,
			healthy_only, token, created_at_ns, updated_at_ns
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name                = excluded.name,
			enabled             = excluded.enabled,
			regex_filters_json  = excluded.regex_filters_json,
			region_filters_json = excluded.region_filters_json,
			healthy_only        = excluded.healthy_only,
			token               = excluded.token,
			updated_at_ns       = excluded.updated_at_ns
	`, p.ID, p.Name, p.Enabled, regexJSON, regionJSON, p.HealthyOnly,
		p.Token, p.CreatedAtNs, p.UpdatedAtNs)
	if err != nil {
		if isSQLiteUniqueConstraint(err) {
			return fmt.Errorf("%w: subscription publication name or token already exists", ErrConflict)
		}
		return err
	}
	return nil
}

// GetSubscriptionPublication 按 ID 读取一条订阅发布配置。
func (r *StateRepo) GetSubscriptionPublication(id string) (*model.SubscriptionPublication, error) {
	row := r.db.QueryRow(`
		SELECT id, name, enabled, regex_filters_json, region_filters_json,
		       healthy_only, token, created_at_ns, updated_at_ns
		FROM subscription_publications
		WHERE id = ?
	`, id)
	p, err := scanSubscriptionPublication(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListSubscriptionPublications 读取全部订阅发布配置。
func (r *StateRepo) ListSubscriptionPublications() ([]model.SubscriptionPublication, error) {
	rows, err := r.db.Query(`
		SELECT id, name, enabled, regex_filters_json, region_filters_json,
		       healthy_only, token, created_at_ns, updated_at_ns
		FROM subscription_publications
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.SubscriptionPublication
	for rows.Next() {
		p, scanErr := scanSubscriptionPublication(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteSubscriptionPublication 按 ID 删除一条订阅发布配置。
func (r *StateRepo) DeleteSubscriptionPublication(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	result, err := r.db.Exec("DELETE FROM subscription_publications WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// scanSubscriptionPublication 扫描一行订阅发布配置并解码过滤规则。
func scanSubscriptionPublication(row rowScanner) (*model.SubscriptionPublication, error) {
	var (
		p          model.SubscriptionPublication
		regexJSON  string
		regionJSON string
	)
	if err := row.Scan(
		&p.ID,
		&p.Name,
		&p.Enabled,
		&regexJSON,
		&regionJSON,
		&p.HealthyOnly,
		&p.Token,
		&p.CreatedAtNs,
		&p.UpdatedAtNs,
	); err != nil {
		return nil, err
	}
	regexFilters, err := decodeStringSliceJSON(regexJSON)
	if err != nil {
		return nil, fmt.Errorf("decode subscription publication %s regex_filters: %w", p.ID, err)
	}
	regionFilters, err := decodeStringSliceJSON(regionJSON)
	if err != nil {
		return nil, fmt.Errorf("decode subscription publication %s region_filters: %w", p.ID, err)
	}
	p.RegexFilters = regexFilters
	p.RegionFilters = regionFilters
	return &p, nil
}
