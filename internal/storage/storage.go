package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

const (
	keyringService = "emir-agent"
	keyringUser    = "agent-credentials"
)

// secureCredentials is the payload stored in the OS credential store.
type secureCredentials struct {
	Token        string `json:"token"`
	PrivateKeyB64 string `json:"private_key_b64"`
}

// SecureTokenStorage abstracts the OS credential store.
type SecureTokenStorage struct {
	service string
	user    string
}

// NewSecureTokenStorage returns a token-backed credential manager.
func NewSecureTokenStorage() *SecureTokenStorage {
	return &SecureTokenStorage{
		service: keyringService,
		user:    keyringUser,
	}
}

// Store saves the agent token and private key in the OS credential store.
func (s *SecureTokenStorage) Store(token string, privateKey []byte) error {
	creds := secureCredentials{
		Token:         token,
		PrivateKeyB64: base64.StdEncoding.EncodeToString(privateKey),
	}
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	return keyring.Set(s.service, s.user, string(data))
}

// Retrieve reads the agent token and private key from the OS credential store.
func (s *SecureTokenStorage) Retrieve() (token string, privateKey []byte, err error) {
	data, err := keyring.Get(s.service, s.user)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil, nil
		}
		return "", nil, err
	}

	var creds secureCredentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return "", nil, fmt.Errorf("unmarshal credentials: %w", err)
	}

	privateKey, err = base64.StdEncoding.DecodeString(creds.PrivateKeyB64)
	if err != nil {
		return "", nil, fmt.Errorf("decode private key: %w", err)
	}

	return creds.Token, privateKey, nil
}

// Delete removes the agent credentials from the OS credential store.
func (s *SecureTokenStorage) Delete() error {
	return keyring.Delete(s.service, s.user)
}

// StateRepository persists non-secret local state to disk.
type StateRepository struct {
	path string
}

// NewStateRepository returns a file-based state repository.
func NewStateRepository(path string) *StateRepository {
	return &StateRepository{path: path}
}

// Load reads the persisted agent state from disk.
func (r *StateRepository) Load() (*models.AgentState, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var state models.AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}
	return &state, nil
}

// Save writes the agent state to disk, creating the directory if needed.
func (r *StateRepository) Save(state *models.AgentState) error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.WriteFile(r.path, data, 0600); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}

// Clear removes both the local state file and the secure token.
func (r *StateRepository) Clear(tokenStore *SecureTokenStorage) error {
	_ = tokenStore.Delete()
	if err := os.Remove(r.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove state file: %w", err)
	}
	return nil
}
