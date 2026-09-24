package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// KeyPair holds an Ed25519 key pair used to sign agent requests.
type KeyPair struct {
	Public  ed25519.PublicKey
	Private ed25519.PrivateKey
}

// GenerateKeyPair creates a new Ed25519 key pair.
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 keys: %w", err)
	}
	return &KeyPair{Public: pub, Private: priv}, nil
}

// KeyPairFromPrivateKey reconstructs a key pair from a private key seed or full key.
func KeyPairFromPrivateKey(privateKey []byte) (*KeyPair, error) {
	if len(privateKey) == ed25519.SeedSize {
		priv := ed25519.NewKeyFromSeed(privateKey)
		return &KeyPair{Public: priv.Public().(ed25519.PublicKey), Private: priv}, nil
	}
	if len(privateKey) == ed25519.PrivateKeySize {
		priv := ed25519.PrivateKey(privateKey)
		return &KeyPair{Public: priv.Public().(ed25519.PublicKey), Private: priv}, nil
	}
	return nil, fmt.Errorf("invalid ed25519 private key length: %d", len(privateKey))
}

// PublicKeyBase64 returns the public key encoded in base64.
func (kp *KeyPair) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(kp.Public)
}

// PrivateKeySeed returns the 32-byte seed of the private key for secure storage.
func (kp *KeyPair) PrivateKeySeed() []byte {
	return kp.Private.Seed()
}

// SignTimestamp signs an RFC3339 timestamp and returns the base64 signature.
func (kp *KeyPair) SignTimestamp(timestamp string) (string, error) {
	signature := ed25519.Sign(kp.Private, []byte(timestamp))
	return base64.StdEncoding.EncodeToString(signature), nil
}

// NowRFC3339 returns the current UTC time in RFC3339 format.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// TokenAuth bundles the agent token and key pair used to authenticate requests.
type TokenAuth struct {
	AgentToken string
	KeyPair    *KeyPair
}

// NewTokenAuth creates an auth bundle from an existing token and key pair.
func NewTokenAuth(agentToken string, keyPair *KeyPair) *TokenAuth {
	return &TokenAuth{AgentToken: agentToken, KeyPair: keyPair}
}

// SignRequest returns the Authorization header value, timestamp, and signature.
func (ta *TokenAuth) SignRequest() (authorization, timestamp, signature string, err error) {
	timestamp = NowRFC3339()
	signature, err = ta.KeyPair.SignTimestamp(timestamp)
	if err != nil {
		return "", "", "", err
	}
	authorization = "Agent " + ta.AgentToken
	return authorization, timestamp, signature, nil
}
