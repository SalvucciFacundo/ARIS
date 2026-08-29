package image

import (
	"fmt"
	"sort"
	"sync"

	"aris/internal/core/ports"
)

var _ ports.BackendRegistry = (*Registry)(nil)

// Registry manages registered image backends.
type Registry struct {
	mu           sync.RWMutex
	backends     map[string]ports.ImageBackend
	defaultName  string
}

// NewRegistry creates a new image backend registry.
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]ports.ImageBackend),
	}
}

// Register adds a new backend to the registry.
func (r *Registry) Register(backend ports.ImageBackend) error {
	if backend == nil {
		return fmt.Errorf("cannot register nil backend")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	name := backend.Name()
	r.backends[name] = backend

	if r.defaultName == "" {
		r.defaultName = name
	}
	return nil
}

// Get returns the backend with the given name.
func (r *Registry) Get(name string) (ports.ImageBackend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backend, exists := r.backends[name]
	if !exists {
		return nil, fmt.Errorf("image backend %q not found (available: %v)", name, r.listUnsafe())
	}
	return backend, nil
}

// List returns sorted names of all registered backends.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.listUnsafe()
}

func (r *Registry) listUnsafe() []string {
	names := make([]string, 0, len(r.backends))
	for name := range r.backends {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SetDefault changes the default backend.
func (r *Registry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.backends[name]; !exists {
		return fmt.Errorf("backend %q is not registered", name)
	}
	r.defaultName = name
	return nil
}

// GetDefault returns the default backend, or nil if none registered.
func (r *Registry) GetDefault() ports.ImageBackend {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.defaultName == "" {
		return nil
	}
	return r.backends[r.defaultName]
}
