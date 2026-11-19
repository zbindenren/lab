// Package tokentest provides the in-memory token store the auth tests drive the
// keychain seam with.
package tokentest

import "github.com/zalando/go-keyring"

// Store is an in-memory token store that satisfies internal.TokenStore. Its
// errors are scriptable, so a test can tell a missing entry apart from a
// keychain that failed. The package does not import internal: internal's own
// tests use it, and the import would be a cycle.
type Store struct {
	Values    map[string]string
	GetErr    error
	SetErr    error
	DeleteErr error
}

// New creates an empty store.
func New() *Store {
	return &Store{Values: map[string]string{}}
}

// Get returns the value stored under service, or keyring.ErrNotFound when there
// is none.
func (s *Store) Get(service string) (string, error) {
	if s.GetErr != nil {
		return "", s.GetErr
	}

	value, ok := s.Values[service]
	if !ok {
		return "", keyring.ErrNotFound
	}

	return value, nil
}

// Set stores value under service.
func (s *Store) Set(service, value string) error {
	if s.SetErr != nil {
		return s.SetErr
	}

	s.Values[service] = value

	return nil
}

// Delete removes the entry stored under service, or reports keyring.ErrNotFound
// when there is none.
func (s *Store) Delete(service string) error {
	if s.DeleteErr != nil {
		return s.DeleteErr
	}

	if _, ok := s.Values[service]; !ok {
		return keyring.ErrNotFound
	}

	delete(s.Values, service)

	return nil
}
