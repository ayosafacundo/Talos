package security

import "testing"

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
