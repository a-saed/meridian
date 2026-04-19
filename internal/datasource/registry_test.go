package datasource_test

import (
	"context"
	"testing"

	"github.com/paulmach/orb"

	"meridian/internal/datasource"
)

// mockSource satisfies DataSource for testing.
type mockSource struct{ name string }

func (m *mockSource) Query(_ context.Context, _ datasource.Query) ([]datasource.Feature, error) {
	return []datasource.Feature{{Geometry: orb.Point{1, 2}}}, nil
}
func (m *mockSource) Close() error { return nil }

func TestRegistryResolve(t *testing.T) {
	reg := datasource.NewRegistry()
	reg.Register("roads", &mockSource{name: "roads"})

	src, ok := reg.Resolve("roads")
	if !ok {
		t.Fatal("expected to resolve 'roads'")
	}
	if src == nil {
		t.Fatal("expected non-nil source")
	}
}

func TestRegistryMiss(t *testing.T) {
	reg := datasource.NewRegistry()
	_, ok := reg.Resolve("nonexistent")
	if ok {
		t.Fatal("expected miss for unregistered layer")
	}
}

func TestRegistryUnregister(t *testing.T) {
	reg := datasource.NewRegistry()
	reg.Register("roads", &mockSource{})
	reg.Unregister("roads")

	_, ok := reg.Resolve("roads")
	if ok {
		t.Fatal("expected miss after unregister")
	}
}
