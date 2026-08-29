package vfs

import "sync"

type memRegistry struct {
	mu         sync.RWMutex
	namespaces map[string]Namespace
	impls      map[string]VFS
}

func (r *memRegistry) Register(ns Namespace, impl VFS) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.namespaces[ns.ID]; exists {
		return ErrNamespaceConflict
	}
	r.namespaces[ns.ID] = ns
	r.impls[ns.ID] = impl
	return nil
}

func (r *memRegistry) Get(namespaceID string) (VFS, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	impl, ok := r.impls[namespaceID]
	return impl, ok
}

func (r *memRegistry) List(callerNamespace string) []Namespace {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Namespace, 0, len(r.namespaces))
	for _, ns := range r.namespaces {
		out = append(out, ns)
	}
	return out
}

func (r *memRegistry) Unregister(namespaceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.namespaces, namespaceID)
	delete(r.impls, namespaceID)
}
