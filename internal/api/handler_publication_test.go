package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Resinat/Resin/internal/node"
	"github.com/Resinat/Resin/internal/service"
	"github.com/Resinat/Resin/internal/subscription"
	"github.com/Resinat/Resin/internal/testutil"
)

func createTestSubscriptionWithNode(t *testing.T, srv *Server, cp *service.ControlPlaneService) string {
	t.Helper()

	createRec := doJSONRequest(t, srv, http.MethodPost, "/api/v1/subscriptions", map[string]any{
		"name": "pubtest",
		"url":  "https://example.com/sub",
	}, true)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create subscription status: got %d body=%s", createRec.Code, createRec.Body.String())
	}
	body := decodeJSONMap(t, createRec)
	subID, _ := body["id"].(string)
	if subID == "" {
		t.Fatalf("subscription id missing: %s", createRec.Body.String())
	}
	sub := cp.SubMgr.Lookup(subID)
	if sub == nil {
		t.Fatalf("subscription %s not found", subID)
	}

	raw := []byte(`{"type":"shadowsocks","tag":"hk-node","server":"198.51.100.10","server_port":8388,"method":"aes-256-gcm","password":"secret"}`)
	hash := node.HashFromRawOptions(raw)
	cp.Pool.AddNodeFromSub(hash, raw, subID)
	sub.ManagedNodes().StoreNode(hash, subscription.ManagedNode{Tags: []string{"hk-node"}})
	entry, ok := cp.Pool.GetEntry(hash)
	if !ok {
		t.Fatalf("node %s not found in pool", hash.Hex())
	}
	outbound := testutil.NewNoopOutbound()
	entry.Outbound.Store(&outbound)
	entry.CircuitOpenSince.Store(0)
	return subID
}

func createTestPublication(t *testing.T, srv *Server) service.SubscriptionPublicationResponse {
	t.Helper()
	rec := doJSONRequest(t, srv, http.MethodPost, "/api/v1/subscription-publications", map[string]any{
		"name":           "ClashHK",
		"enabled":        true,
		"regex_filters":  []string{"hk"},
		"region_filters": []string{},
		"healthy_only":   false,
	}, true)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create publication status: got %d body=%s", rec.Code, rec.Body.String())
	}
	var created service.SubscriptionPublicationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode publication: %v", err)
	}
	return created
}

func TestHandleSubscriptionPublicationLifecycle(t *testing.T) {
	srv, cp, _ := newControlPlaneTestServer(t)
	createTestSubscriptionWithNode(t, srv, cp)
	created := createTestPublication(t, srv)

	if created.NodeCount != 1 || created.MatchingNodeCount != 1 {
		t.Fatalf("created stats = %+v", created)
	}
	if !strings.HasSuffix(created.SubscriptionPath, ".yaml") {
		t.Fatalf("subscription_path = %q", created.SubscriptionPath)
	}
	if !strings.Contains(created.ProviderPath, "format=provider") {
		t.Fatalf("provider_path = %q", created.ProviderPath)
	}
	if !strings.HasSuffix(created.SingboxPath, ".json") {
		t.Fatalf("singbox_path = %q", created.SingboxPath)
	}

	listRec := doJSONRequest(t, srv, http.MethodGet, "/api/v1/subscription-publications", nil, true)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status: got %d body=%s", listRec.Code, listRec.Body.String())
	}
	var page PageResponse[service.SubscriptionPublicationResponse]
	if err := json.Unmarshal(listRec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("page = %+v", page)
	}

	getRec := doJSONRequest(t, srv, http.MethodGet, "/api/v1/subscription-publications/"+created.ID, nil, true)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status: got %d body=%s", getRec.Code, getRec.Body.String())
	}

	deleteRec := doJSONRequest(t, srv, http.MethodDelete, "/api/v1/subscription-publications/"+created.ID, nil, true)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status: got %d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestHandleSubscriptionPublicationContent(t *testing.T) {
	srv, cp, _ := newControlPlaneTestServer(t)
	createTestSubscriptionWithNode(t, srv, cp)
	created := createTestPublication(t, srv)

	clashRec := doJSONRequest(t, srv, http.MethodGet, created.SubscriptionPath, nil, false)
	if clashRec.Code != http.StatusOK {
		t.Fatalf("clash status: got %d body=%s", clashRec.Code, clashRec.Body.String())
	}
	if got := clashRec.Header().Get("X-Resin-Node-Count"); got != "1" {
		t.Fatalf("X-Resin-Node-Count = %q", got)
	}
	clashBody := clashRec.Body.String()
	if !strings.Contains(clashBody, "proxies:") || !strings.Contains(clashBody, "198.51.100.10") {
		t.Fatalf("unexpected clash body:\n%s", clashBody)
	}
	if !strings.Contains(clashBody, "proxy-groups:") {
		t.Fatalf("full config missing proxy-groups:\n%s", clashBody)
	}

	providerRec := doJSONRequest(t, srv, http.MethodGet, created.ProviderPath, nil, false)
	if providerRec.Code != http.StatusOK {
		t.Fatalf("provider status: got %d body=%s", providerRec.Code, providerRec.Body.String())
	}
	providerBody := providerRec.Body.String()
	if strings.Contains(providerBody, "proxy-groups:") {
		t.Fatalf("provider body should not contain proxy-groups:\n%s", providerBody)
	}

	singboxRec := doJSONRequest(t, srv, http.MethodGet, created.SingboxPath, nil, false)
	if singboxRec.Code != http.StatusOK {
		t.Fatalf("singbox status: got %d body=%s", singboxRec.Code, singboxRec.Body.String())
	}
	var singbox map[string]any
	if err := json.Unmarshal(singboxRec.Body.Bytes(), &singbox); err != nil {
		t.Fatalf("decode singbox: %v body=%s", err, singboxRec.Body.String())
	}
	if _, ok := singbox["outbounds"]; !ok {
		t.Fatalf("singbox body missing outbounds: %s", singboxRec.Body.String())
	}
}

func TestHandleSubscriptionPublicationRejectsBadTokenAndDisabled(t *testing.T) {
	srv, cp, _ := newControlPlaneTestServer(t)
	createTestSubscriptionWithNode(t, srv, cp)
	created := createTestPublication(t, srv)

	badPath := strings.Replace(created.SubscriptionPath, created.ID+"/", created.ID+"/wrong-token", 1)
	badRec := doJSONRequest(t, srv, http.MethodGet, badPath, nil, false)
	if badRec.Code != http.StatusNotFound {
		t.Fatalf("bad token status = %d, want 404", badRec.Code)
	}

	updateRec := doJSONRequest(t, srv, http.MethodPatch, "/api/v1/subscription-publications/"+created.ID, map[string]any{
		"enabled": false,
	}, true)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("disable status: got %d body=%s", updateRec.Code, updateRec.Body.String())
	}
	disabledRec := doJSONRequest(t, srv, http.MethodGet, created.SubscriptionPath, nil, false)
	if disabledRec.Code != http.StatusNotFound {
		t.Fatalf("disabled status = %d, want 404", disabledRec.Code)
	}
}

func TestHandleRotateSubscriptionPublicationToken(t *testing.T) {
	srv, cp, _ := newControlPlaneTestServer(t)
	createTestSubscriptionWithNode(t, srv, cp)
	created := createTestPublication(t, srv)

	rotateRec := doJSONRequest(t, srv, http.MethodPost, "/api/v1/subscription-publications/"+created.ID+"/actions/rotate-token", nil, true)
	if rotateRec.Code != http.StatusOK {
		t.Fatalf("rotate status: got %d body=%s", rotateRec.Code, rotateRec.Body.String())
	}
	var rotated service.SubscriptionPublicationResponse
	if err := json.Unmarshal(rotateRec.Body.Bytes(), &rotated); err != nil {
		t.Fatalf("decode rotated: %v", err)
	}
	if rotated.SubscriptionPath == created.SubscriptionPath {
		t.Fatal("subscription path did not change after rotation")
	}

	oldRec := doJSONRequest(t, srv, http.MethodGet, created.SubscriptionPath, nil, false)
	if oldRec.Code != http.StatusNotFound {
		t.Fatalf("old token status = %d, want 404", oldRec.Code)
	}
	newRec := doJSONRequest(t, srv, http.MethodGet, rotated.SubscriptionPath, nil, false)
	if newRec.Code != http.StatusOK {
		t.Fatalf("new token status = %d body=%s", newRec.Code, newRec.Body.String())
	}
}
