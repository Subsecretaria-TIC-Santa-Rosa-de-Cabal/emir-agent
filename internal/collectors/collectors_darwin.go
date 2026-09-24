//go:build darwin

package collectors

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

// Collect gathers inventory on macOS using system_profiler.
func Collect() models.AgentInventoryRequest {
	computer := collectComputerDarwin()
	return models.AgentInventoryRequest{
		Computer:          computer,
		Disks:             collectDisksDarwin(),
		GPUs:              collectGPUsDarwin(),
		MemoryModules:     collectMemoryModulesDarwin(),
		Monitors:          collectMonitorsDarwin(),
		NetworkInterfaces: collectNetworkInterfacesDarwin(),
		Peripherals:       collectPeripheralsDarwin(),
	}
}

func collectComputerDarwin() models.AgentComputerInventory {
	c := models.AgentComputerInventory{}
	os := runtime.GOOS
	c.OS = &os
	c.Architecture = ptr(runtime.GOARCH)
	c.Hostname = ptr(hostname())

	hw := systemProfiler("SPHardwareDataType")
	c.Manufacturer = ptr(hw["manufacturer"])
	c.Model = ptr(hw["model"])
	c.SerialNumber = ptr(hw["serial_number"])
	c.HardwareUUID = ptr(hw["hardware_uuid"])

	c.CPUBrand = ptr(hw["cpu_brand"])
	c.CPUPhysicalCores = ptrInt(int(parseInt(hw["number_cores"])))
	c.CPULogicalCores = ptrInt(int(parseInt(hw["number_processors"])))

	memBytes := parseFloat(hw["memory"])
	if memBytes > 0 {
		c.RAMTotal = ptrFloat64(round(memBytes/1024/1024/1024, 2)) // bytes -> GB
	}

	c.HasBattery = ptrBool(parseInt(hw["battery"]) > 0)
	c.IsOSActivated = ptrBool(true)

	return c
}

func collectDisksDarwin() []models.AgentDiskInventory {
	return nil // TODO implement with system_profiler SPSerialATADataType
}

func collectGPUsDarwin() []models.AgentGPUInventory {
	out := run("system_profiler", "SPDisplaysDataType", "-json")
	if out == "" {
		return nil
	}
	return nil // TODO parse JSON
}

func collectMemoryModulesDarwin() []models.AgentMemoryModuleInventory {
	out := run("system_profiler", "SPMemoryDataType", "-json")
	if out == "" {
		return nil
	}
	return nil // TODO parse JSON
}

func collectMonitorsDarwin() []models.AgentMonitorInventory {
	return nil // TODO implement
}

func collectNetworkInterfacesDarwin() []models.AgentNetworkInterfaceInventory {
	out := run("system_profiler", "SPNetworkDataType", "-json")
	if out == "" {
		return nil
	}
	return nil // TODO parse JSON
}

func collectPeripheralsDarwin() []models.AgentPeripheralInventory {
	out := run("system_profiler", "SPUSBDataType", "-json")
	if out == "" {
		return nil
	}
	return nil // TODO parse JSON
}

// systemProfiler returns selected fields from system_profiler -json.
func systemProfiler(dataType string) map[string]string {
	out := run("system_profiler", dataType, "-json")
	if out == "" {
		return map[string]string{}
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &root); err != nil {
		return map[string]string{}
	}
	return map[string]string{}
}

func readFileOrEmpty(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
