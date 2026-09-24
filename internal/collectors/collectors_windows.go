//go:build windows

package collectors

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/emir/emir-agent/internal/models"
)

// Collect gathers full inventory on Windows using PowerShell and WMI.
func Collect() models.AgentInventoryRequest {
	computer := collectComputerWindows()
	return models.AgentInventoryRequest{
		Computer:          computer,
		Disks:             collectDisksWindows(),
		GPUs:              collectGPUsWindows(),
		MemoryModules:     collectMemoryModulesWindows(),
		Monitors:          collectMonitorsWindows(),
		NetworkInterfaces: collectNetworkInterfacesWindows(),
		Peripherals:       collectPeripheralsWindows(),
	}
}

func collectComputerWindows() models.AgentComputerInventory {
	c := models.AgentComputerInventory{}

	os := runtime.GOOS
	c.OS = &os
	c.Architecture = ptr(runtime.GOARCH)
	c.Hostname = ptr(hostname())

	c.Manufacturer = ptr(ps("Get-CimInstance Win32_ComputerSystem | Select-Object -ExpandProperty Manufacturer"))
	c.Model = ptr(ps("Get-CimInstance Win32_ComputerSystem | Select-Object -ExpandProperty Model"))
	c.SerialNumber = ptr(ps("Get-CimInstance Win32_BIOS | Select-Object -ExpandProperty SerialNumber"))

	uuid := strings.Trim(ps("Get-CimInstance Win32_ComputerSystemProduct | Select-Object -ExpandProperty UUID"), "{}")
	c.HardwareUUID = ptr(uuid)

	c.MotherboardManufacturer = ptr(ps("Get-CimInstance Win32_BaseBoard | Select-Object -ExpandProperty Manufacturer"))
	c.MotherboardModel = ptr(ps("Get-CimInstance Win32_BaseBoard | Select-Object -ExpandProperty Product"))
	c.MotherboardSerialNumber = ptr(ps("Get-CimInstance Win32_BaseBoard | Select-Object -ExpandProperty SerialNumber"))

	c.BIOSManufacturer = ptr(ps("Get-CimInstance Win32_BIOS | Select-Object -ExpandProperty Manufacturer"))
	c.BIOSVersion = ptr(ps("Get-CimInstance Win32_BIOS | Select-Object -ExpandProperty Version"))
	c.BIOSSerialNumber = ptr(ps("Get-CimInstance Win32_BIOS | Select-Object -ExpandProperty SerialNumber"))

	c.CPUBrand = ptr(ps("Get-CimInstance Win32_Processor | Select-Object -First 1 -ExpandProperty Name"))
	c.CPUPhysicalCores = ptrInt(int(parseInt(ps("Get-CimInstance Win32_Processor | Select-Object -First 1 -ExpandProperty NumberOfCores"))))
	c.CPULogicalCores = ptrInt(int(parseInt(ps("Get-CimInstance Win32_Processor | Select-Object -First 1 -ExpandProperty NumberOfLogicalProcessors"))))
	c.CPUFrequency = parseFloatPtr(ps("Get-CimInstance Win32_Processor | Select-Object -First 1 -ExpandProperty MaxClockSpeed"))
	if c.CPUFrequency != nil {
		*c.CPUFrequency /= 1000 // MHz -> GHz
	}

	memBytes := parseFloat(ps("Get-CimInstance Win32_ComputerSystem | Select-Object -ExpandProperty TotalPhysicalMemory"))
	if memBytes > 0 {
		c.RAMTotal = ptrFloat64(round(memBytes / 1024 / 1024 / 1024, 2)) // bytes -> GB
	}

	battery := ps("Get-CimInstance Win32_Battery | Select-Object -First 1 -ExpandProperty EstimatedChargeRemaining")
	if battery != "" {
		c.HasBattery = ptrBool(true)
		c.BatteryCapacity = parseFloatPtr(battery)
	} else {
		c.HasBattery = ptrBool(false)
	}

	c.IsOSActivated = ptrBool(windowsActivated())
	c.OfficeSuite, c.IsOfficeActivated = officeInfoWindows()
	c.AntivirusName = ptr(antivirusWindows())
	c.IsWazuhConfigured = ptrBool(serviceExistsWindows("Wazuh"))
	c.RustdeskID = ptr(rustdeskIDWindows())

	return c
}

func collectDisksWindows() []models.AgentDiskInventory {
	script := `
Get-CimInstance Win32_DiskDrive | ForEach-Object {
    [PSCustomObject]@{
        capacity = [math]::Round($_.Size / 1GB, 2)
        type = if ($_.MediaType -like "*SSD*") { "SSD" } elseif ($_.MediaType -like "*HDD*" -or $_.MediaType -like "*Fixed*") { "HDD" } else { "UNKNOWN" }
        manufacturer = $_.Manufacturer
        model = $_.Model
        serial_number = $_.SerialNumber
        interface = $_.InterfaceType
    }
} | ConvertTo-Json -Compress
`
	out := ps(script)
	if out == "" {
		return nil
	}
	var disks []models.AgentDiskInventory
	if err := unmarshalJsonArray(out, &disks); err != nil {
		return nil
	}
	return disks
}

func collectGPUsWindows() []models.AgentGPUInventory {
	script := `
Get-CimInstance Win32_VideoController | ForEach-Object {
    [PSCustomObject]@{
        manufacturer = $_.AdapterCompatibility
        model = $_.Name
        memory = if ($_.AdapterRAM -and $_.AdapterRAM -gt 0) { [math]::Round($_.AdapterRAM / 1MB, 2) } else { $null }
    }
} | ConvertTo-Json -Compress
`
	out := ps(script)
	if out == "" {
		return nil
	}
	var gpus []models.AgentGPUInventory
	if err := unmarshalJsonArray(out, &gpus); err != nil {
		return nil
	}
	return gpus
}

func collectMemoryModulesWindows() []models.AgentMemoryModuleInventory {
	script := `
Get-CimInstance Win32_PhysicalMemory | ForEach-Object {
    [PSCustomObject]@{
        capacity = [math]::Round($_.Capacity / 1MB, 2)
        type = $_.MemoryType
        speed = $_.Speed
        manufacturer = $_.Manufacturer
        serial_number = $_.SerialNumber
        model = $_.PartNumber
        slot = $_.DeviceLocator
    }
} | ConvertTo-Json -Compress
`
	out := ps(script)
	if out == "" {
		return nil
	}
	var modules []models.AgentMemoryModuleInventory
	if err := unmarshalJsonArray(out, &modules); err != nil {
		return nil
	}
	return modules
}

func collectMonitorsWindows() []models.AgentMonitorInventory {
	script := `
Get-CimInstance WmiMonitorBasicDisplayParams | ForEach-Object {
    [PSCustomObject]@{
        manufacturer = $_.InstanceName
        model = $_.InstanceName
        serial_number = $null
        screen = $null
        resolution_width = $_.MaxHorizontalImageSize
        resolution_height = $_.MaxVerticalImageSize
        refresh_rate = $null
        is_primary = $null
    }
} | ConvertTo-Json -Compress
`
	out := ps(script)
	if out == "" {
		return nil
	}
	var monitors []models.AgentMonitorInventory
	if err := unmarshalJsonArray(out, &monitors); err != nil {
		return nil
	}
	return monitors
}

func collectNetworkInterfacesWindows() []models.AgentNetworkInterfaceInventory {
	script := `
Get-CimInstance Win32_NetworkAdapterConfiguration -Filter "IPEnabled='True'" | ForEach-Object {
    $nic = Get-CimInstance Win32_NetworkAdapter | Where-Object { $_.Index -eq $_.Index }
    [PSCustomObject]@{
        type = $_.Description
        mac_address = $_.MACAddress
        ipv4 = ($_.IPAddress | Where-Object { $_ -like "*.*" } | Select-Object -First 1)
        ipv6 = ($_.IPAddress | Where-Object { $_ -like "*:*" } | Select-Object -First 1)
        speed = if ($nic.Speed) { [math]::Round($nic.Speed / 1000000, 2) } else { $null }
        dns = ($_.DNSServerSearchOrder -join ", ")
        gateway = ($_.DefaultIPGateway -join ", ")
        rustdesk_id = $null
        rustdesk_key = $null
        is_active_directory_joined = (Get-CimInstance Win32_ComputerSystem).PartOfDomain
        is_currently_connected = $_.IPEnabled
    }
} | ConvertTo-Json -Compress
`
	out := ps(script)
	if out == "" {
		return nil
	}
	var interfaces []models.AgentNetworkInterfaceInventory
	if err := unmarshalJsonArray(out, &interfaces); err != nil {
		return nil
	}
	return interfaces
}

func collectPeripheralsWindows() []models.AgentPeripheralInventory {
	script := `
Get-CimInstance Win32_PnPEntity | Where-Object { $_.Status -eq "OK" -and $_.PNPClass -eq "USB" } | ForEach-Object {
    [PSCustomObject]@{
        name = $_.Name
        model = $_.Name
        serial_number = $null
        tag = $null
    }
} | ConvertTo-Json -Compress
`
	out := ps(script)
	if out == "" {
		return nil
	}
	var peripherals []models.AgentPeripheralInventory
	if err := unmarshalJsonArray(out, &peripherals); err != nil {
		return nil
	}
	return peripherals
}

// ps runs a PowerShell command and returns trimmed stdout.
func ps(script string) string {
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func unmarshalJsonArray(data string, target any) error {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}
	if !strings.HasPrefix(data, "[") {
		data = "[" + data + "]"
	}
	return json.Unmarshal([]byte(data), target)
}

func windowsActivated() bool {
	out := ps("(Get-CimInstance SoftwareLicensingProduct | Where-Object { $_.PartialProductKey -and $_.Name -like '*Windows*' } | Select-Object -First 1 -ExpandProperty LicenseStatus)")
	return parseInt(out) == 1
}

func officeInfoWindows() (*string, *bool) {
	out := ps("Get-CimInstance SoftwareLicensingProduct | Where-Object { $_.Name -like '*Office*' } | Select-Object -First 1 Name, LicenseStatus | ConvertTo-Json -Compress")
	if out == "" {
		return nil, nil
	}
	var item struct {
		Name          string `json:"Name"`
		LicenseStatus int    `json:"LicenseStatus"`
	}
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return nil, nil
	}
	return &item.Name, ptrBool(item.LicenseStatus == 1)
}

func antivirusWindows() string {
	return ps("Get-CimInstance AntiVirusProduct | Select-Object -First 1 -ExpandProperty displayName")
}

func serviceExistsWindows(name string) bool {
	out := ps(fmt.Sprintf("Get-Service -Name \"%s*\" -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty Name", name))
	return out != ""
}

func rustdeskIDWindows() string {
	return ps("Get-ItemProperty -Path 'HKCU:\\Software\\RustDesk' -Name 'id' -ErrorAction SilentlyContinue | Select-Object -ExpandProperty id")
}
