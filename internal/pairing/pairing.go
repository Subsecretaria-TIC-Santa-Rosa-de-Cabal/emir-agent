package pairing

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/auth"
	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

// Prompt asks the technician for core URL and pairing code.
func Prompt(defaultCoreURL string) (coreURL, pairingCode string, err error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("EMIR Core URL [%s]: ", defaultCoreURL)
	coreURL, err = reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("read core url: %w", err)
	}
	coreURL = strings.TrimSpace(coreURL)
	if coreURL == "" {
		coreURL = defaultCoreURL
	}

	fmt.Print("Pairing code: ")
	pairingCode, err = reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("read pairing code: %w", err)
	}
	pairingCode = strings.TrimSpace(pairingCode)
	if pairingCode == "" {
		return "", "", fmt.Errorf("pairing code is required")
	}

	return coreURL, pairingCode, nil
}

// BuildPairRequest creates a pair request using the generated key pair and local identifiers.
func BuildPairRequest(pairingCode string, keyPair *auth.KeyPair, hostname, serialNumber, hardwareUUID *string) models.AgentPairRequest {
	return models.AgentPairRequest{
		PairingCode:  pairingCode,
		PublicKey:    keyPair.PublicKeyBase64(),
		Hostname:     hostname,
		SerialNumber: serialNumber,
		HardwareUUID: hardwareUUID,
	}
}
