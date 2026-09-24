//go:build linux

package collectors

import (
	"bufio"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models"
)

// Collect gathers inventory on Linux using /proc, /sys and common CLI tools.
func Collect() models.AgentInventoryRequest {
	computer := collectComputerLinux()
	return models.AgentInventoryRequest{
		Computer:          computer,
		Disks:             collectDisksLinux(),
		GPUs:              collectGPUsLinux(),
		MemoryModules:     collectMemoryModulesLinux(),
		Monitors:          collectMonitorsLinux(),
		NetworkInterfaces: collectNetworkInterfacesLinux(),
		Peripherals:       collectPeripheralsLinux(),
	}
}

func collectComputerLinux() models.AgentComputerInventory {
	c := models.AgentComputerInventory{}
	os := runtime.GOOS
	c.OS = &os
	c.Architecture = ptr(runtime.GOARCH)
	c.Hostname = ptr(hostname())

	c.Manufacturer = ptr(run("dmidecode", "-s", "system-manufacturer"))
	c.Model = ptr(run("dmidecode", "-s", "system-product-name"))
	c.SerialNumber = ptr(run("dmidecode", "-s", "system-serial-number"))
	c.HardwareUUID = ptr(run("dmidecode", "-s", "system-uuid"))

	c.MotherboardManufacturer = ptr(run("dmidecode", "-s", "baseboard-manufacturer"))
	c.MotherboardModel = ptr(run("dmidecode", "-s", "baseboard-product-name"))
	c.MotherboardSerialNumber = ptr(run("dmidecode", "-s", "baseboard-serial-number"))

	c.BIOSManufacturer = ptr(run("dmidecode", "-s", "bios-vendor"))
	c.BIOSVersion = ptr(run("dmidecode", "-s", "bios-version"))

	c.CPUBrand = ptr(cpuBrandLinux())
	c.CPUPhysicalCores = ptrInt(int(parseInt(run("bash", "-c", "nproc --all"))))
	c.CPULogicalCores = c.CPUPhysicalCores
	c.CPUFrequency = parseFloatPtr(run("bash", "-c", "lscpu | grep 'CPU max MHz' | awk '{print $NF}'"))
	if c.CPUFrequency != nil {
		*c.CPUFrequency /= 1000
	}

	memKB := parseFloat(run("bash", "-c", "grep MemTotal /proc/meminfo | awk '{print $2}'"))
	if memKB > 0 {
		c.RAMTotal = ptrFloat64(round(memKB/1024/1024, 2)) // KB -> GB
	}

	c.HasBattery = ptrBool(batteryExistsLinux())
	c.IsOSActivated = ptrBool(true) // Linux activation concept differs

	return c
}

func collectDisksLinux() []models.AgentDiskInventory {
	out := run("lsblk", "-Jbo", "NAME,SIZE,TYPE,MODEL,SERIAL,TRAN")
	if out == "" {
		return nil
	}
	return nil // TODO parse lsblk JSON
}

func collectGPUsLinux() []models.AgentGPUInventory {
	out := run("lspci")
	if out == "" {
		return nil
	}
	var gpus []models.AgentGPUInventory
	re := regexp.MustCompile(`VGA compatible controller:\s*(.+)`)
	for _, line := range strings.Split(out, "\n") {
		if m := re.FindStringSubmatch(line); m != nil {
			gpus = append(gpus, models.AgentGPUInventory{Model: ptr(strings.TrimSpace(m[1]))})
		}
	}
	return gpus
}

func collectMemoryModulesLinux() []models.AgentMemoryModuleInventory {
	// dmidecode requires root; fallback to a single summary module.
	memKB := parseFloat(run("bash", "-c", "grep MemTotal /proc/meminfo | awk '{print $2}'"))
	if memKB == 0 {
		return nil
	}
	return []models.AgentMemoryModuleInventory{
		{
			Capacity: round(memKB/1024, 2), // MB
			Type:     "DDR",
			Speed:    0,
		},
	}
}

func collectMonitorsLinux() []models.AgentMonitorInventory {
	// Minimal implementation; xrandr required.
	return nil
}

func collectNetworkInterfacesLinux() []models.AgentNetworkInterfaceInventory {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil
	}
	defer file.Close()

	var interfaces []models.AgentNetworkInterfaceInventory
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "|") || strings.Contains(line, "face") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name == "lo" {
			continue
		}
		interfaces = append(interfaces, models.AgentNetworkInterfaceInventory{
			Type:               name,
			MACAddress:         ptr(run("bash", "-c", "cat /sys/class/net/"+name+"/address")),
			IsCurrentlyConnected: ptrBool(true),
		})
	}
	return interfaces
}

func collectPeripheralsLinux() []models.AgentPeripheralInventory {
	out := run("lsusb")
	if out == "" {
		return nil
	}
	var peripherals []models.AgentPeripheralInventory
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, " ", 7)
		if len(parts) < 7 {
			continue
		}
		name := strings.TrimSpace(parts[6])
		if name == "" {
			continue
		}
		peripherals = append(peripherals, models.AgentPeripheralInventory{Name: name})
	}
	return peripherals
}

func cpuBrandLinux() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func batteryExistsLinux() bool {
	_, err := os.Stat("/sys/class/power_supply/BAT0")
	return err == nil
}
