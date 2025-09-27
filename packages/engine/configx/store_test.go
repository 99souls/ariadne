package configx

import (
	"testing"
)

func TestStoreAppendAndHead(t *testing.T) {
	store := NewVersionedStore()
	if _, ok := store.Head(); ok { t.Fatalf("expected empty store head") }
	_, err := store.Append(&EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 3}}, "actor1", "init", 0)
	if err != nil { t.Fatalf("append v1 failed: %v", err) }
	head, ok := store.Head()
	if !ok || head.Version != 1 { t.Fatalf("expected head version 1") }
	if head.Hash == "" { t.Fatalf("expected hash to be set") }

	_, err = store.Append(&EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 5}}, "actor2", "bump concurrency", 1)
	if err != nil { t.Fatalf("append v2 failed: %v", err) }
	head, _ = store.Head()
	if head.Version != 2 { t.Fatalf("expected head version 2") }
	if head.Parent != 1 { t.Fatalf("expected parent=1 got %d", head.Parent) }
}

func TestStoreParentMismatch(t *testing.T) {
	store := NewVersionedStore()
	_, _ = store.Append(&EngineConfigSpec{}, "a", "init", 0)
	_, err := store.Append(&EngineConfigSpec{}, "b", "second", 999) // wrong parent expectation
	if err == nil { t.Fatalf("expected parent mismatch error") }
}

func TestStoreVerify(t *testing.T) {
	store := NewVersionedStore()
	vc, err := store.Append(&EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 1}}, "a", "init", 0)
	if err != nil { t.Fatalf("append failed: %v", err) }
	if err := store.Verify(vc.Version); err != nil { t.Fatalf("verify failed: %v", err) }
}

func TestAuditImmutability(t *testing.T) {
	store := NewVersionedStore()
	_, _ = store.Append(&EngineConfigSpec{Policies: &PoliciesConfigSection{EnabledFlags: map[string]bool{"x": true}}}, "a", "init", 0)
	records := store.ListAudit()
	if len(records) != 1 { t.Fatalf("expected 1 record") }
	records[0].Actor = "mutated" // should not affect internal state
	again := store.ListAudit()
	if again[0].Actor == "mutated" { t.Fatalf("audit slice not copied (mutation leaked)") }
}
