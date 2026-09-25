package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

// secureCredentials is the payload persisted to disk.
type secureCredentials struct {
	Token         string `json:"token"`
	PrivateKeyB64 string `json:"private_key_b64"`
}

// SecureTokenStorage persists agent credentials to a file in the same
// directory as the local state file. This makes credentials accessible to
// the service account (e.g., Windows SYSTEM) that runs the agent, avoiding
// per-user OS credential store limitations.
type SecureTokenStorage struct {
	path string
}

// NewSecureTokenStorage returns a file-backed credential manager.
// statePath is used to derive the credentials file location.
func NewSecureTokenStorage(statePath string) *SecureTokenStorage {
	return &SecureTokenStorage{
		path: filepath.Join(filepath.Dir(statePath), "credentials.json"),
	}
}

// Store saves the agent token and private key to disk.
func (s *SecureTokenStorage) Store(token string, privateKey []byte) error {
	creds := secureCredentials{
		Token:         token,
		PrivateKeyB64: base64.StdEncoding.EncodeToString(privateKey),
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create credentials dir: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("write credentials file: %w", err)
	}
	return nil
}

// Retrieve reads the agent token and private key from disk.
func (s *SecureTokenStorage) Retrieve() (token string, privateKey []byte, err error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil
		}
		return "", nil, fmt.Errorf("read credentials file: %w", err)
	}

	var creds secureCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", nil, fmt.Errorf("unmarshal credentials: %w", err)
	}

	privateKey, err = base64.StdEncoding.DecodeString(creds.PrivateKeyB64)
	if err != nil {
		return "", nil, fmt.Errorf("decode private key: %w", err)
	}

	return creds.Token, privateKey, nil
}

// Delete removes the persisted credentials file.
func (s *SecureTokenStorage) Delete() error {
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove credentials file: %w", err)
	}
	return nil
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

// Clear removes both the local state file and the secure token file.
func (r *StateRepository) Clear(tokenStore *SecureTokenStorage) error {
	_ = tokenStore.Delete()
	if err := os.Remove(r.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove state file: %w", err)
	}
	return nil
}
