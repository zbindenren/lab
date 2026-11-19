package internal

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	// TokenEnvVar is the environment variable holding the GitLab token. It
	// outranks every other token source and is read directly, not through the
	// config file, whose values are expanded from the environment before
	// parsing and can therefore no longer be attributed to a source.
	TokenEnvVar = "GITLAB_TOKEN"

	// keychainAccount is the account of a lab keychain entry. It is empty to keep
	// the same entry shape as glab for the same host.
	keychainAccount = ""

	// #nosec G101 -- the message only names the environment variable, it is not a credential
	noTokenMessage = "no GitLab token: set GITLAB_TOKEN, run `lab auth login`, or set gitlab.token via `lab config`"
)

// ErrNoToken reports that every token source missed, which is the only
// resolution failure that means "no token anywhere". Callers therefore tell it
// apart from a source that failed, which is a credential problem of its own.
var ErrNoToken = errors.New(noTokenMessage)

// TokenSource is the source the active GitLab token came from.
type TokenSource string

const (
	// TokenSourceEnv means the token came from the GITLAB_TOKEN environment variable.
	TokenSourceEnv TokenSource = "env"
	// TokenSourceKeychain means the token came from the OS keychain.
	TokenSourceKeychain TokenSource = "keychain"
	// TokenSourceConfigFile means the token came from gitlab.token in the config file.
	TokenSourceConfigFile TokenSource = "config file"
	// TokenSourceNone means no token was found.
	TokenSourceNone TokenSource = "none"
)

// TokenStore is the OS keychain boundary. A missing entry must return an error
// satisfying errors.Is(err, keyring.ErrNotFound); every other error is reported
// to the user instead of falling back to the next token source.
type TokenStore interface {
	Get(service string) (string, error)
	Set(service, value string) error
	Delete(service string) error
}

// KeychainStore is the TokenStore backed by the OS keychain.
type KeychainStore struct{}

// NewKeychainStore creates a token store backed by the OS keychain.
func NewKeychainStore() *KeychainStore {
	return &KeychainStore{}
}

// Get returns the token stored under service.
func (s *KeychainStore) Get(service string) (string, error) {
	return keyring.Get(service, keychainAccount)
}

// Set stores value under service.
func (s *KeychainStore) Set(service, value string) error {
	return keyring.Set(service, keychainAccount, value)
}

// Delete removes the entry stored under service.
func (s *KeychainStore) Delete(service string) error {
	return keyring.Delete(service, keychainAccount)
}

// TokenResolver resolves the GitLab token from the environment, the OS keychain
// and the config file, in that order.
type TokenResolver struct {
	store TokenStore
	env   func(string) string
}

// NewTokenResolver creates a resolver that reads the OS keychain through store and
// the environment through env, which must look the variable up directly rather
// than through the config file.
func NewTokenResolver(store TokenStore, env func(string) string) *TokenResolver {
	return &TokenResolver{store: store, env: env}
}

// Resolve returns the token of the GitLab instance behind baseURL and the source
// it came from. configToken is the gitlab.token value of the config file, which
// ranks below the keychain: storing a token in the keychain must not require
// editing a config file that already carries one.
func (r *TokenResolver) Resolve(baseURL, configToken string) (string, TokenSource, error) {
	if token := r.env(TokenEnvVar); token != "" {
		return token, TokenSourceEnv, nil
	}

	token, err := r.fromKeychain(baseURL)
	if err != nil {
		return "", TokenSourceNone, err
	}

	if token != "" {
		return token, TokenSourceKeychain, nil
	}

	if configToken != "" {
		return configToken, TokenSourceConfigFile, nil
	}

	return "", TokenSourceNone, ErrNoToken
}

// fromKeychain returns the stored token and an empty string when the entry is
// missing or empty, so that a missing entry can be told apart from a keychain
// failure, which is returned instead of falling back to the config file.
func (r *TokenResolver) fromKeychain(baseURL string) (string, error) {
	service := KeychainService(baseURL)

	token, err := r.store.Get(service)
	if err != nil {
		// only a missing entry falls through to the config file, so a locked or
		// denied keychain cannot be mistaken for "no token stored".
		if errors.Is(err, keyring.ErrNotFound) {
			return "", nil
		}

		return "", fmt.Errorf("failed to read the GitLab token from the OS keychain (%s): %w", service, err)
	}

	return token, nil
}

// GitLabHost returns the host of the GitLab instance behind baseURL: lowercased,
// without scheme or path, keeping an explicit port. It normalises through the
// same function the config uses, so the host of a stored base URL and the host of
// a hand-written one cannot disagree.
func GitLabHost(baseURL string) string {
	u, err := url.Parse(normalizeBaseURL(strings.TrimSpace(baseURL)))
	if err != nil {
		return ""
	}

	return strings.ToLower(u.Host)
}

// KeychainService returns the keychain service a token of the GitLab instance
// behind baseURL is stored under.
func KeychainService(baseURL string) string {
	return "lab:" + GitLabHost(baseURL)
}
