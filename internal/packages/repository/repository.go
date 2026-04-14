package repository

import "context"

// Descriptor is a remote catalog entry (future: curated registry / store).
type Descriptor struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source,omitempty"`
	InstallURL string `json:"install_url,omitempty"`
}

// Catalog lists installable remote packages.
type Catalog interface {
	List(ctx context.Context) ([]Descriptor, error)
}

// Stub returns an empty catalog until a real registry is wired.
type Stub struct{}

// NewStub builds a no-op catalog implementation.
func NewStub() *Stub {
	return &Stub{}
}

// List implements Catalog.
func (Stub) List(ctx context.Context) ([]Descriptor, error) {
	_ = ctx
	return []Descriptor{}, nil
}
