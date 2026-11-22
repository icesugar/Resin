package api

import (
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/Resinat/Resin/internal/service"
)

func publicationMatchesKeyword(p service.SubscriptionPublicationResponse, keyword string) bool {
	contains := func(v string) bool {
		return strings.Contains(strings.ToLower(v), keyword)
	}
	if contains(p.ID) || contains(p.Name) {
		return true
	}
	for _, item := range p.RegexFilters {
		if contains(item) {
			return true
		}
	}
	for _, item := range p.RegionFilters {
		if contains(item) {
			return true
		}
	}
	return false
}

func filterPublicationsByKeyword(items []service.SubscriptionPublicationResponse, rawKeyword string) []service.SubscriptionPublicationResponse {
	keyword := strings.ToLower(strings.TrimSpace(rawKeyword))
	if keyword == "" {
		return items
	}
	filtered := make([]service.SubscriptionPublicationResponse, 0, len(items))
	for _, item := range items {
		if publicationMatchesKeyword(item, keyword) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func publicationSortKey(sortBy string, p service.SubscriptionPublicationResponse) string {
	switch sortBy {
	case "created_at":
		return p.CreatedAt
	case "updated_at":
		return p.UpdatedAt
	default:
		return p.Name
	}
}

// HandleListSubscriptionPublications 处理 GET /api/v1/subscription-publications。
func HandleListSubscriptionPublications(cp *service.ControlPlaneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		enabled, ok := parseBoolQueryOrWriteInvalid(w, r, "enabled")
		if !ok {
			return
		}
		items, err := cp.ListSubscriptionPublications(enabled)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		items = filterPublicationsByKeyword(items, r.URL.Query().Get("keyword"))

		sorting, ok := parseSortingOrWriteInvalid(
			w,
			r,
			[]string{"name", "created_at", "updated_at"},
			"name",
			"asc",
		)
		if !ok {
			return
		}
		SortSlice(items, sorting, func(item service.SubscriptionPublicationResponse) string {
			return publicationSortKey(sorting.SortBy, item)
		})

		pg, ok := parsePaginationOrWriteInvalid(w, r)
		if !ok {
			return
		}
		WritePage(w, http.StatusOK, items, pg)
	}
}

// HandleGetSubscriptionPublication 处理 GET /api/v1/subscription-publications/{id}。
func HandleGetSubscriptionPublication(cp *service.ControlPlaneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := requireUUIDPathParam(w, r, "id", "publication_id")
		if !ok {
			return
		}
		item, err := cp.GetSubscriptionPublication(id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, item)
	}
}

// HandleCreateSubscriptionPublication 处理 POST /api/v1/subscription-publications。
func HandleCreateSubscriptionPublication(cp *service.ControlPlaneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req service.CreateSubscriptionPublicationRequest
		if err := DecodeBody(r, &req); err != nil {
			writeDecodeBodyError(w, err)
			return
		}
		item, err := cp.CreateSubscriptionPublication(req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusCreated, item)
	}
}

// HandleUpdateSubscriptionPublication 处理 PATCH /api/v1/subscription-publications/{id}。
func HandleUpdateSubscriptionPublication(cp *service.ControlPlaneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := requireUUIDPathParam(w, r, "id", "publication_id")
		if !ok {
			return
		}
		var req service.UpdateSubscriptionPublicationRequest
		if err := DecodeBody(r, &req); err != nil {
			writeDecodeBodyError(w, err)
			return
		}
		item, err := cp.UpdateSubscriptionPublication(id, req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, item)
	}
}

// HandleDeleteSubscriptionPublication 处理 DELETE /api/v1/subscription-publications/{id}。
func HandleDeleteSubscriptionPublication(cp *service.ControlPlaneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := requireUUIDPathParam(w, r, "id", "publication_id")
		if !ok {
			return
		}
		if err := cp.DeleteSubscriptionPublication(id); err != nil {
			writeServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// HandleRotateSubscriptionPublicationToken 处理 POST /api/v1/subscription-publications/{id}/actions/rotate-token。
func HandleRotateSubscriptionPublicationToken(cp *service.ControlPlaneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := requireUUIDPathParam(w, r, "id", "publication_id")
		if !ok {
			return
		}
		item, err := cp.RotateSubscriptionPublicationToken(id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, item)
	}
}

// HandlePreviewSubscriptionPublication 处理 POST /api/v1/subscription-publications/preview。
func HandlePreviewSubscriptionPublication(cp *service.ControlPlaneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req service.PreviewSubscriptionPublicationRequest
		if err := DecodeBody(r, &req); err != nil {
			writeDecodeBodyError(w, err)
			return
		}
		result, err := cp.PreviewSubscriptionPublication(req)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, result)
	}
}

// HandleGetSubscriptionPublicationContent 是公开订阅路由，不经过 admin token 认证。
func HandleGetSubscriptionPublicationContent(cp *service.ControlPlaneService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := PathParam(r, "id")
		if !ValidateUUID(id) {
			http.NotFound(w, r)
			return
		}

		file := PathParam(r, "file")
		token := file
		target := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("target")))
		lowerFile := strings.ToLower(file)
		switch {
		case strings.HasSuffix(lowerFile, ".yaml"):
			token = file[:len(file)-len(".yaml")]
			if target == "" {
				target = "clash"
			}
		case strings.HasSuffix(lowerFile, ".yml"):
			token = file[:len(file)-len(".yml")]
			if target == "" {
				target = "clash"
			}
		case strings.HasSuffix(lowerFile, ".json"):
			token = file[:len(file)-len(".json")]
			if target == "" {
				target = "singbox"
			}
		}
		if token == "" {
			http.NotFound(w, r)
			return
		}
		if target == "" {
			target = "clash"
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" {
			format = "config"
		}

		content, err := cp.RenderSubscriptionPublicationContent(id, token, target, format)
		if err != nil {
			var svcErr *service.ServiceError
			if errors.As(err, &svcErr) {
				switch svcErr.Code {
				case "NOT_FOUND":
					http.NotFound(w, r)
					return
				case "INVALID_ARGUMENT":
					http.Error(w, svcErr.Message, http.StatusBadRequest)
					return
				}
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		disposition := mime.FormatMediaType("attachment", map[string]string{"filename": content.Filename})
		w.Header().Set("Content-Type", content.ContentType)
		w.Header().Set("Content-Disposition", disposition)
		w.Header().Set("Cache-Control", "private, max-age=60")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("profile-update-interval", "24")
		w.Header().Set("X-Resin-Node-Count", strconv.Itoa(content.NodeCount))
		w.Header().Set("X-Resin-Skipped-Nodes", strconv.Itoa(content.Skipped))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content.Content)
	}
}
