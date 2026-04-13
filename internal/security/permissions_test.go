package security

import (
	"sync"
	"testing"
	"time"
)

func TestPermissionsClear(t *testing.T) {
	t.Parallel()

	p := NewPermissions(func(string, string, string) (bool, string) {
		return false, "no"
	})
	p.Set("app.a", "fs:external", true)
	if !p.IsGranted("app.a", "fs:external") {
		t.Fatal("expected granted")
	}
	p.Clear("app.a", "fs:external")
	if p.IsGranted("app.a", "fs:external") {
		t.Fatal("expected cleared scope to be not granted")
	}
}

func TestRequestBlocksUntilCompletePendingDecision(t *testing.T) {
	t.Parallel()

	p := NewPermissions(func(string, string, string) (bool, string) {
		return false, MsgPendingHostApproval
	})

	var wg sync.WaitGroup
	wg.Add(1)
	var granted bool
	var msg string
	go func() {
		defer wg.Done()
		granted, msg = p.Request("app.x", "net:out", "because")
	}()

	time.Sleep(50 * time.Millisecond)
	p.CompletePendingDecision("app.x", "net:out", true, "ok")

	wg.Wait()
	if !granted || msg != "ok" {
		t.Fatalf("expected granted ok, got granted=%v msg=%q", granted, msg)
	}
	if !p.IsGranted("app.x", "net:out") {
		t.Fatal("expected scope granted after wait")
	}
}
