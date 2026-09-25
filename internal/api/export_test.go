package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"workbuddy2api-gui/internal/authstore"
	"workbuddy2api-gui/internal/ops"
)

func TestAccountExportRoutes(t *testing.T) {
	dir := t.TempDir()
	store, err := authstore.New(dir)
	if err != nil {
		t.Fatalf("authstore.New: %v", err)
	}

	a := &authstore.Account{
		UID:          "acc-test-1",
		Nickname:     "测试号",
		AccessToken:  "token-xyz",
		RefreshToken: "refresh-xyz",
		Domain:       "copilot.tencent.com",
		ExpiresAt:    1800000000,
	}
	if err := store.Save(a); err != nil {
		t.Fatalf("save: %v", err)
	}

	cfg := testConfig()
	svc := ops.New(cfg, store, nil, nil)
	srv := NewServer(cfg, svc, http.NotFoundHandler(), "test")

	// 模拟登录以拿到有效 cookie
	token, ok, _ := srv.sessions.Create("admin", "secret123", cfg, "127.0.0.1")
	if !ok {
		t.Fatal("login failed")
	}

	handler := srv.Handler()

	// 1. 测试导出凭证路由
	{
		req := httptest.NewRequest(http.MethodPost, "/api/accounts/export/credentials", strings.NewReader(`{}`))
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("export/credentials HTTP %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			OK       bool                       `json:"ok"`
			Count    int                        `json:"count"`
			Format   string                     `json:"format"`
			Exported []ops.CredentialExportItem `json:"exported"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !resp.OK || resp.Count != 1 || len(resp.Exported) != 1 {
			t.Errorf("export/credentials payload mismatch: %+v", resp)
		}
		if resp.Exported[0].UID != "acc-test-1" {
			t.Errorf("exported UID got %q want acc-test-1", resp.Exported[0].UID)
		}
	}

	// 2. 测试导出清单路由
	{
		req := httptest.NewRequest(http.MethodPost, "/api/accounts/export/inventory", strings.NewReader(`{}`))
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("export/inventory HTTP %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			OK       bool                       `json:"ok"`
			Count    int                        `json:"count"`
			Format   string                     `json:"format"`
			Exported []ops.AccountInventoryItem `json:"exported"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !resp.OK || resp.Count != 1 || len(resp.Exported) != 1 {
			t.Errorf("export/inventory payload mismatch: %+v", resp)
		}
		if resp.Exported[0].UID != "acc-test-1" || resp.Exported[0].Realm != "cn" {
			t.Errorf("exported item mismatch: %+v", resp.Exported[0])
		}
	}
}
