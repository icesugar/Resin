package service

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Resinat/Resin/internal/model"
	"github.com/Resinat/Resin/internal/node"
	"github.com/Resinat/Resin/internal/platform"
	"github.com/Resinat/Resin/internal/publish"
	"github.com/Resinat/Resin/internal/state"
)

// SubscriptionPublicationResponse 是订阅发布配置的 API 响应。
type SubscriptionPublicationResponse struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Enabled           bool     `json:"enabled"`
	RegexFilters      []string `json:"regex_filters"`
	RegionFilters     []string `json:"region_filters"`
	HealthyOnly       bool     `json:"healthy_only"`
	MatchingNodeCount int      `json:"matching_node_count"`
	NodeCount         int      `json:"node_count"`
	SkippedNodeCount  int      `json:"skipped_node_count"`
	HealthyNodeCount  int      `json:"healthy_node_count"`
	SubscriptionPath  string   `json:"subscription_path"`
	ProviderPath      string   `json:"provider_path"`
	SingboxPath       string   `json:"singbox_path"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
}

// SubscriptionPublicationContent 是公开订阅路由返回的内容。
type SubscriptionPublicationContent struct {
	Content     []byte
	ContentType string
	Filename    string
	NodeCount   int
	Skipped     int
}

// CreateSubscriptionPublicationRequest 是创建订阅发布的请求体。
type CreateSubscriptionPublicationRequest struct {
	Name          *string  `json:"name"`
	Enabled       *bool    `json:"enabled"`
	RegexFilters  []string `json:"regex_filters"`
	RegionFilters []string `json:"region_filters"`
	HealthyOnly   *bool    `json:"healthy_only"`
}

// UpdateSubscriptionPublicationRequest 是更新订阅发布的请求体。
type UpdateSubscriptionPublicationRequest struct {
	Name          *string   `json:"name"`
	Enabled       *bool     `json:"enabled"`
	RegexFilters  *[]string `json:"regex_filters"`
	RegionFilters *[]string `json:"region_filters"`
	HealthyOnly   *bool     `json:"healthy_only"`
}

// PreviewSubscriptionPublicationRequest 是预览过滤规则的请求体。
type PreviewSubscriptionPublicationRequest struct {
	RegexFilters  []string `json:"regex_filters"`
	RegionFilters []string `json:"region_filters"`
	HealthyOnly   *bool    `json:"healthy_only"`
}

// PreviewSubscriptionPublicationResponse 是过滤规则预览结果。
type PreviewSubscriptionPublicationResponse struct {
	MatchingNodeCount int      `json:"matching_node_count"`
	NodeCount         int      `json:"node_count"`
	SkippedNodeCount  int      `json:"skipped_node_count"`
	HealthyNodeCount  int      `json:"healthy_node_count"`
	SampleNames       []string `json:"sample_names"`
}

type publicationNodeSet struct {
	Nodes        []publish.Node
	Matching     int
	Healthy      int
	Exportable   int
	Skipped      int
	SkippedTypes map[string]int
}

func publicationPaths(id, token string) (clash, provider, singbox string) {
	base := "/sub/" + id + "/" + token
	return base + ".yaml", base + ".yaml?format=provider", base + ".json"
}

func publicationToResponse(p model.SubscriptionPublication) SubscriptionPublicationResponse {
	clashPath, providerPath, singboxPath := publicationPaths(p.ID, p.Token)
	return SubscriptionPublicationResponse{
		ID:               p.ID,
		Name:             p.Name,
		Enabled:          p.Enabled,
		RegexFilters:     append([]string(nil), p.RegexFilters...),
		RegionFilters:    append([]string(nil), p.RegionFilters...),
		HealthyOnly:      p.HealthyOnly,
		SubscriptionPath: clashPath,
		ProviderPath:     providerPath,
		SingboxPath:      singboxPath,
		CreatedAt:        time.Unix(0, p.CreatedAtNs).UTC().Format(time.RFC3339Nano),
		UpdatedAt:        time.Unix(0, p.UpdatedAtNs).UTC().Format(time.RFC3339Nano),
	}
}

// ListSubscriptionPublications 返回全部订阅发布配置。
func (s *ControlPlaneService) ListSubscriptionPublications(enabled *bool) ([]SubscriptionPublicationResponse, error) {
	publications, err := s.Engine.ListSubscriptionPublications()
	if err != nil {
		return nil, internal("list subscription publications", err)
	}
	result := make([]SubscriptionPublicationResponse, 0, len(publications))
	for _, p := range publications {
		if enabled != nil && p.Enabled != *enabled {
			continue
		}
		result = append(result, s.withPublicationStats(publicationToResponse(p), p))
	}
	return result, nil
}

// GetSubscriptionPublication 返回单条订阅发布配置。
func (s *ControlPlaneService) GetSubscriptionPublication(id string) (*SubscriptionPublicationResponse, error) {
	p, err := s.getPublicationModel(id)
	if err != nil {
		return nil, err
	}
	resp := s.withPublicationStats(publicationToResponse(*p), *p)
	return &resp, nil
}

// CreateSubscriptionPublication 创建订阅发布配置。
func (s *ControlPlaneService) CreateSubscriptionPublication(req CreateSubscriptionPublicationRequest) (*SubscriptionPublicationResponse, error) {
	if req.Name == nil {
		return nil, invalidArg("name is required")
	}
	name := platform.NormalizePlatformName(*req.Name)
	if name == "" {
		return nil, invalidArg("name is required")
	}
	if err := platform.ValidatePlatformName(name); err != nil {
		return nil, invalidArg("name: " + err.Error())
	}

	regexFilters := append([]string(nil), req.RegexFilters...)
	regionFilters := append([]string(nil), req.RegionFilters...)
	if err := validatePublicationFilters(regexFilters, regionFilters); err != nil {
		return nil, err
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	healthyOnly := false
	if req.HealthyOnly != nil {
		healthyOnly = *req.HealthyOnly
	}

	token, err := publish.GenerateToken()
	if err != nil {
		return nil, internal("generate publication token", err)
	}
	now := time.Now().UnixNano()
	p := model.SubscriptionPublication{
		ID:            uuid.New().String(),
		Name:          name,
		Enabled:       enabled,
		RegexFilters:  regexFilters,
		RegionFilters: regionFilters,
		HealthyOnly:   healthyOnly,
		Token:         token,
		CreatedAtNs:   now,
		UpdatedAtNs:   now,
	}
	if err := s.Engine.UpsertSubscriptionPublication(p); err != nil {
		if errors.Is(err, state.ErrConflict) {
			return nil, conflict("subscription publication name already exists")
		}
		return nil, internal("persist subscription publication", err)
	}

	resp := s.withPublicationStats(publicationToResponse(p), p)
	return &resp, nil
}

// UpdateSubscriptionPublication 局部更新订阅发布配置。
func (s *ControlPlaneService) UpdateSubscriptionPublication(id string, req UpdateSubscriptionPublicationRequest) (*SubscriptionPublicationResponse, error) {
	if req.Name == nil && req.Enabled == nil && req.RegexFilters == nil &&
		req.RegionFilters == nil && req.HealthyOnly == nil {
		return nil, invalidArg("empty patch")
	}

	p, err := s.getPublicationModel(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := platform.NormalizePlatformName(*req.Name)
		if name == "" {
			return nil, invalidArg("name is required")
		}
		if err := platform.ValidatePlatformName(name); err != nil {
			return nil, invalidArg("name: " + err.Error())
		}
		p.Name = name
	}
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}
	if req.RegexFilters != nil {
		p.RegexFilters = append([]string(nil), (*req.RegexFilters)...)
	}
	if req.RegionFilters != nil {
		p.RegionFilters = append([]string(nil), (*req.RegionFilters)...)
	}
	if req.HealthyOnly != nil {
		p.HealthyOnly = *req.HealthyOnly
	}
	if err := validatePublicationFilters(p.RegexFilters, p.RegionFilters); err != nil {
		return nil, err
	}

	p.UpdatedAtNs = time.Now().UnixNano()
	if err := s.Engine.UpsertSubscriptionPublication(*p); err != nil {
		if errors.Is(err, state.ErrConflict) {
			return nil, conflict("subscription publication name already exists")
		}
		return nil, internal("persist subscription publication", err)
	}

	resp := s.withPublicationStats(publicationToResponse(*p), *p)
	return &resp, nil
}

// DeleteSubscriptionPublication 删除订阅发布配置。
func (s *ControlPlaneService) DeleteSubscriptionPublication(id string) error {
	if err := s.Engine.DeleteSubscriptionPublication(id); err != nil {
		if errors.Is(err, state.ErrNotFound) {
			return notFound("subscription publication not found")
		}
		return internal("delete subscription publication", err)
	}
	return nil
}

// RotateSubscriptionPublicationToken 重新生成订阅访问 token。
func (s *ControlPlaneService) RotateSubscriptionPublicationToken(id string) (*SubscriptionPublicationResponse, error) {
	p, err := s.getPublicationModel(id)
	if err != nil {
		return nil, err
	}
	token, err := publish.GenerateToken()
	if err != nil {
		return nil, internal("generate publication token", err)
	}
	p.Token = token
	p.UpdatedAtNs = time.Now().UnixNano()
	if err := s.Engine.UpsertSubscriptionPublication(*p); err != nil {
		if errors.Is(err, state.ErrConflict) {
			return nil, conflict("subscription publication token conflict")
		}
		return nil, internal("persist subscription publication", err)
	}
	resp := s.withPublicationStats(publicationToResponse(*p), *p)
	return &resp, nil
}

// PreviewSubscriptionPublication 预览过滤规则命中的节点。
func (s *ControlPlaneService) PreviewSubscriptionPublication(req PreviewSubscriptionPublicationRequest) (*PreviewSubscriptionPublicationResponse, error) {
	regexFilters := append([]string(nil), req.RegexFilters...)
	regionFilters := append([]string(nil), req.RegionFilters...)
	healthyOnly := false
	if req.HealthyOnly != nil {
		healthyOnly = *req.HealthyOnly
	}

	nodes, err := s.collectPublicationNodes(regexFilters, regionFilters, healthyOnly)
	if err != nil {
		return nil, err
	}

	samples := make([]string, 0, 20)
	for _, n := range nodes.Nodes {
		if len(samples) >= 20 {
			break
		}
		samples = append(samples, n.Name)
	}
	if samples == nil {
		samples = []string{}
	}
	return &PreviewSubscriptionPublicationResponse{
		MatchingNodeCount: nodes.Matching,
		NodeCount:         nodes.Exportable,
		SkippedNodeCount:  nodes.Skipped,
		HealthyNodeCount:  nodes.Healthy,
		SampleNames:       samples,
	}, nil
}

// RenderSubscriptionPublicationContent 校验 token 并生成订阅内容。
func (s *ControlPlaneService) RenderSubscriptionPublicationContent(id, token, target, format string) (*SubscriptionPublicationContent, error) {
	p, err := s.getPublicationModel(id)
	if err != nil {
		return nil, err
	}
	if !p.Enabled {
		return nil, notFound("subscription publication not found")
	}
	if subtle.ConstantTimeCompare([]byte(p.Token), []byte(token)) != 1 {
		return nil, notFound("subscription publication not found")
	}

	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		target = "clash"
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "config"
	}

	nodes, err := s.collectPublicationNodes(p.RegexFilters, p.RegionFilters, p.HealthyOnly)
	if err != nil {
		return nil, err
	}

	var (
		result      publish.RenderResult
		contentType string
		extension   string
	)
	switch target {
	case "clash":
		if format != "config" && format != "provider" {
			return nil, invalidArg("format: must be config or provider")
		}
		result, err = publish.RenderClash(nodes.Nodes, publish.ClashOptions{
			GroupName:    p.Name,
			ProviderOnly: format == "provider",
		})
		contentType = "text/yaml; charset=utf-8"
		extension = ".yaml"
	case "singbox":
		if format != "config" {
			return nil, invalidArg("format: singbox only supports config")
		}
		result, err = publish.RenderSingbox(nodes.Nodes, publish.SingboxOptions{GroupName: p.Name})
		contentType = "application/json; charset=utf-8"
		extension = ".json"
	default:
		return nil, invalidArg("target: must be clash or singbox")
	}
	if err != nil {
		return nil, internal("render subscription publication", err)
	}

	filename := publish.SanitizeFilename(p.Name, "subscription") + extension
	return &SubscriptionPublicationContent{
		Content:     result.Content,
		ContentType: contentType,
		Filename:    filename,
		NodeCount:   result.NodeCount,
		Skipped:     result.Skipped,
	}, nil
}

func (s *ControlPlaneService) getPublicationModel(id string) (*model.SubscriptionPublication, error) {
	p, err := s.Engine.GetSubscriptionPublication(id)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			return nil, notFound("subscription publication not found")
		}
		return nil, internal("get subscription publication", err)
	}
	return p, nil
}

func validatePublicationFilters(regexFilters, regionFilters []string) *ServiceError {
	if _, err := platform.CompileRegexFilters(regexFilters); err != nil {
		return invalidArg(err.Error())
	}
	if err := platform.ValidateRegionFilters(regionFilters); err != nil {
		return invalidArg(err.Error())
	}
	return nil
}

func (s *ControlPlaneService) collectPublicationNodes(
	regexFilters []string,
	regionFilters []string,
	healthyOnly bool,
) (publicationNodeSet, error) {
	tagFilter, err := platform.CompileRegexFilters(regexFilters)
	if err != nil {
		return publicationNodeSet{}, invalidArg(err.Error())
	}
	if err := platform.ValidateRegionFilters(regionFilters); err != nil {
		return publicationNodeSet{}, invalidArg(err.Error())
	}

	result := publicationNodeSet{SkippedTypes: map[string]int{}}
	if s == nil || s.Pool == nil {
		return result, nil
	}

	subLookup := s.Pool.MakeSubLookup()
	var geoLookup platform.GeoLookupFunc
	if s.GeoIP != nil {
		geoLookup = s.GeoIP.Lookup
	}

	s.Pool.Range(func(h node.Hash, entry *node.NodeEntry) bool {
		if entry == nil {
			return true
		}
		if entry.IsDisabledBySubscriptions(subLookup) {
			return true
		}
		if !entry.MatchTagFilter(tagFilter, subLookup) {
			return true
		}
		if len(regionFilters) > 0 {
			region := entry.GetRegion(geoLookup)
			if !platform.MatchRegionFilter(region, regionFilters) {
				return true
			}
		}

		result.Matching++
		healthy := entry.IsHealthy()
		if healthy {
			result.Healthy++
		}
		if healthyOnly && !healthy {
			return true
		}

		result.Nodes = append(result.Nodes, publish.Node{
			Hash:       h.Hex(),
			Name:       s.Pool.ResolveNodeDisplayTag(h),
			RawOptions: append(json.RawMessage(nil), entry.RawOptions...),
		})
		return true
	})

	stats := publish.Analyze(result.Nodes)
	result.Exportable = stats.Exportable
	result.Skipped = stats.Skipped
	result.SkippedTypes = stats.SkippedTypes
	return result, nil
}

func (s *ControlPlaneService) withPublicationStats(
	resp SubscriptionPublicationResponse,
	p model.SubscriptionPublication,
) SubscriptionPublicationResponse {
	nodes, err := s.collectPublicationNodes(p.RegexFilters, p.RegionFilters, p.HealthyOnly)
	if err != nil {
		return resp
	}
	resp.MatchingNodeCount = nodes.Matching
	resp.NodeCount = nodes.Exportable
	resp.SkippedNodeCount = nodes.Skipped
	resp.HealthyNodeCount = nodes.Healthy
	return resp
}
