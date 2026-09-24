//go:build !windows && !linux && !darwin

package collectors

import (
	"runtime"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

// Collect gathers inventory for unsupported platforms returning a minimal set.
func Collect() models.AgentInventoryRequest {
	os := runtime.GOOS
	arch := runtime.GOARCH
	return models.AgentInventoryRequest{
		Computer: models.AgentComputerInventory{
			OS:           &os,
			Architecture: &arch,
		},
	}
}
