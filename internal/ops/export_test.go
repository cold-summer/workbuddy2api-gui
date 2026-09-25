package ops

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"workbuddy2api-gui/internal/authstore"
	"workbuddy2api-gui/internal/config"
)

func TestExportCredentialsAndInventory(t *testing.T) {
	dir := t.TempDir()
	store, err := authstore.New(dir)
	if err != nil {
		t.Fatalf("authstore.New: %v", err)
	}

	// 准备两个账号凭证文件
	a1 := &authstore.Account{
		UID:          "test-uid-1",
		Nickname:     "用户一",
		AccessToken:  "token-1",
		RefreshToken: "refresh-1",
		Domain:       "copilot.tencent.com",
		ExpiresAt:    1800000000,
	}
	a2 := &authstore.Account{
		UID:          "test-uid-2",
		Nickname:     "用户二",
		AccessToken:  "token-2",
		RefreshToken: "refresh-2",
		Domain:       "www.workbuddy.ai",
		ExpiresAt:    1800000000,
	}
	if err := store.Save(a1); err != nil {
		t.Fatalf("save a1: %v", err)
	}
	if err := store.Save(a2); err != nil {
		t.Fatalf("save a2: %v", err)
	}

	// 混入一个非 JSON 损坏文件，验证容错
	_ = os.WriteFile(filepath.Join(dir, "workbuddy-broken.json"), []byte("not-json"), 0o600)

	svc := &Service{
		cfg:   &config.Config{},
		store: store,
	}

	// 1. 测试全部导出凭证
	credsAll, err := svc.ExportCredentials(nil)
	if err != nil {
		t.Fatalf("ExportCredentials(all): %v", err)
	}
	if len(credsAll) != 2 {
		t.Errorf("credsAll len=%d want 2", len(credsAll))
	}

	// 2. 测试指定 UID 导出
	credsOne, err := svc.ExportCredentials([]string{"test-uid-2"})
	if err != nil {
		t.Fatalf("ExportCredentials(test-uid-2): %v", err)
	}
	if len(credsOne) != 1 || credsOne[0].UID != "test-uid-2" {
		t.Errorf("credsOne got %+v want test-uid-2", credsOne)
	}

	// 3. 测试只读模式下拒绝导出敏感凭证
	svcReadOnly := &Service{
		cfg:   &config.Config{ReadOnly: true},
		store: store,
	}
	if _, err := svcReadOnly.ExportCredentials(nil); err == nil {
		t.Errorf("expected error in read-only mode, got nil")
	}

	// 4. 测试账号清单导出（不包含 token，只读模式依然允许）
	inv, err := svcReadOnly.ExportInventory(context.Background(), nil)
	if err != nil {
		t.Fatalf("ExportInventory: %v", err)
	}
	if len(inv) < 2 {
		t.Errorf("inv len=%d want >=2", len(inv))
	}
	// 验证域识别
	foundGlobal, foundCN := false, false
	for _, item := range inv {
		if item.UID == "test-uid-1" && item.Realm == "cn" {
			foundCN = true
		}
		if item.UID == "test-uid-2" && item.Realm == "global" {
			foundGlobal = true
		}
	}
	if !foundCN || !foundGlobal {
		t.Errorf("realm deduction failed: cn=%v global=%v", foundCN, foundGlobal)
	}
}
