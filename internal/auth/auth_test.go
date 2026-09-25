package auth

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if kp.Public == nil || kp.Private == nil {
		t.Fatal("expected public and private keys to be set")
	}
}

func TestPublicKeyBase64(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	b64 := kp.PublicKeyBase64()
	if b64 == "" {
		t.Fatal("expected non-empty base64 public key")
	}
}

func TestSignTimestamp(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature, err := kp.SignTimestamp(timestamp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if signature == "" {
		t.Fatal("expected non-empty signature")
	}
}

func TestSignRequest(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	auth := NewTokenAuth("test-token", kp)
	authorization, timestamp, signature, err := auth.SignRequest()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.HasPrefix(authorization, "Agent test-token") {
		t.Fatalf("expected authorization to start with 'Agent test-token', got %s", authorization)
	}
	if timestamp == "" {
		t.Fatal("expected non-empty timestamp")
	}
	if signature == "" {
		t.Fatal("expected non-empty signature")
	}
}

func TestKeyPairFromPrivateKeySeed(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	seed := kp.PrivateKeySeed()
	restored, err := KeyPairFromPrivateKey(seed)
	if err != nil {
		t.Fatalf("expected no error restoring key pair, got %v", err)
	}
	if restored.PublicKeyBase64() != kp.PublicKeyBase64() {
		t.Fatal("restored public key does not match original")
	}
}

func TestKeyPairFromFullPrivateKey(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	restored, err := KeyPairFromPrivateKey(kp.Private)
	if err != nil {
		t.Fatalf("expected no error restoring key pair, got %v", err)
	}
	if restored.PublicKeyBase64() != kp.PublicKeyBase64() {
		t.Fatal("restored public key does not match original")
	}
}

func TestKeyPairFromInvalidPrivateKey(t *testing.T) {
	_, err := KeyPairFromPrivateKey([]byte("short"))
	if err == nil {
		t.Fatal("expected error for invalid private key length")
	}
}
