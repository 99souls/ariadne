package configx

import (
	"sync"
	"testing"
)

// TestStoreHashCorruption ensures Verify detects tampering.
func TestStoreHashCorruption(t *testing.T) {
	s := NewVersionedStore()
	spec := &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 2}}
	vc, err := s.Append(spec, "actor", "", 0)
	if err != nil {
		t.Fatalf("append err: %v", err)
	}
	if err := s.Verify(vc.Version); err != nil {
		t.Fatalf("verify failed unexpectedly: %v", err)
	}
	// Tamper internal hash (tests share package so can access)
	s.mu.Lock()
	s.versions[vc.Version-1].Hash = "deadbeef"
	s.mu.Unlock()
	if err := s.Verify(vc.Version); err == nil || err != ErrHashMismatch {
		t.Fatalf("expected hash mismatch, got %v", err)
	}
}

// TestApplyNilSpec validates error on nil candidate.
func TestApplyNilSpec(t *testing.T) {
	s := NewVersionedStore()
	a := NewApplier(s, NewSimulator())
	_, err := a.Apply(nil, nil, ApplyOptions{Actor: "x"})
	if err == nil {
		t.Fatalf("expected error for nil spec")
	}
}

// TestRollbackNonexistent ensures error when version not found.
func TestRollbackNonexistent(t *testing.T) {
	s := NewVersionedStore()
	a := NewApplier(s, NewSimulator())
	if _, err := a.Rollback(99, "actor"); err == nil {
		t.Fatalf("expected error rollback nonexistent version")
	}
}

// TestConcurrentAccess performs concurrent reads/appends to surface potential race (use go test -race externally).
func TestConcurrentAccess(t *testing.T) {
	s := NewVersionedStore()
	a := NewApplier(s, NewSimulator())
	base := &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 1}}
	if _, err := a.Apply(nil, base, ApplyOptions{Actor: "init"}); err != nil {
		t.Fatalf("init apply err: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
				spec := &EngineConfigSpec{Global: &GlobalConfigSection{MaxConcurrency: 1 + i}}
				_, _ = a.Apply(nil, spec, ApplyOptions{Actor: "c"}) // ignore errors (simulation always acceptable)
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Head()
			s.Get(1)
		}()
	}
	wg.Wait()
}
