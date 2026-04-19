package datasource

import "sync"

// Registry maps WMS layer names to their DataSource instances.
// It is safe for concurrent read/write.
type Registry struct {
	mu      sync.RWMutex
	sources map[string]DataSource
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{sources: make(map[string]DataSource)}
}

// Register associates name with src. Overwrites any existing entry.
func (r *Registry) Register(name string, src DataSource) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[name] = src
}

// Unregister removes name from the registry and returns the DataSource that was there (or nil).
func (r *Registry) Unregister(name string) DataSource {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.sources[name]
	delete(r.sources, name)
	return old
}

// Swap registers src under name and returns the previous DataSource (or nil).
// Use this instead of separate Unregister+Register to avoid a window with no entry.
func (r *Registry) Swap(name string, src DataSource) DataSource {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.sources[name]
	r.sources[name] = src
	return old
}

// Resolve looks up the DataSource for name.
func (r *Registry) Resolve(name string) (DataSource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src, ok := r.sources[name]
	return src, ok
}
