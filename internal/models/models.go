package models

// Version is the current agent version.
const Version = "0.1.0"

// AgentComputerInventory mirrors emir-core AgentComputerInventory.
type AgentComputerInventory struct {
	Hostname                 *string `json:"hostname,omitempty"`
	OS                       *string `json:"os,omitempty"`
	OSVersion                *string `json:"os_version,omitempty"`
	Architecture             *string `json:"architecture,omitempty"`
	Manufacturer             *string `json:"manufacturer,omitempty"`
	Model                    *string `json:"model,omitempty"`
	SerialNumber             *string `json:"serial_number,omitempty"`
	HardwareUUID             *string `json:"hardware_uuid,omitempty"`
	MotherboardManufacturer  *string `json:"motherboard_manufacturer,omitempty"`
	MotherboardModel         *string `json:"motherboard_model,omitempty"`
	MotherboardSerialNumber  *string `json:"motherboard_serial_number,omitempty"`
	BIOSManufacturer         *string `json:"bios_manufacturer,omitempty"`
	BIOSVersion              *string `json:"bios_version,omitempty"`
	BIOSSerialNumber         *string `json:"bios_serial_number,omitempty"`
	CPUBrand                 *string `json:"cpu_brand,omitempty"`
	CPUPhysicalCores         *int    `json:"cpu_physical_cores,omitempty"`
	CPULogicalCores          *int    `json:"cpu_logical_cores,omitempty"`
	CPUFrequency             *float64 `json:"cpu_frequency,omitempty"`
	RAMTotal                 *float64 `json:"ram_total,omitempty"`
	HasBattery               *bool   `json:"has_battery,omitempty"`
	BatteryCapacity          *float64 `json:"battery_capacity,omitempty"`
	IsOSActivated            *bool   `json:"is_os_activated,omitempty"`
	OfficeSuite              *string `json:"office_suite,omitempty"`
	IsOfficeActivated        *bool   `json:"is_office_activated,omitempty"`
	OfficeTag                *string `json:"office_tag,omitempty"`
	OfficeActivation         *string `json:"office_activation,omitempty"`
	AntivirusName            *string `json:"antivirus_name,omitempty"`
	RustdeskID               *string `json:"rustdesk_id,omitempty"`
	RustdeskPassword         *string `json:"rustdesk_password,omitempty"`
	IsWazuhConfigured        *bool   `json:"is_wazuh_configured,omitempty"`
}

// AgentDiskInventory mirrors emir-core AgentDiskInventory.
type AgentDiskInventory struct {
	Capacity     float64 `json:"capacity"`
	Type         string  `json:"type"`
	Manufacturer *string `json:"manufacturer,omitempty"`
	Model        *string `json:"model,omitempty"`
	SerialNumber *string `json:"serial_number,omitempty"`
	Interface    *string `json:"interface,omitempty"`
}

// AgentGPUInventory mirrors emir-core AgentGPUInventory.
type AgentGPUInventory struct {
	Manufacturer *string `json:"manufacturer,omitempty"`
	Model        *string `json:"model,omitempty"`
	Memory       *float64 `json:"memory,omitempty"`
}

// AgentMemoryModuleInventory mirrors emir-core AgentMemoryModuleInventory.
type AgentMemoryModuleInventory struct {
	Capacity     float64 `json:"capacity"`
	Type         string  `json:"type"`
	Speed        float64 `json:"speed"`
	Manufacturer *string `json:"manufacturer,omitempty"`
	SerialNumber *string `json:"serial_number,omitempty"`
	Model        *string `json:"model,omitempty"`
	Slot         *string `json:"slot,omitempty"`
}

// AgentMonitorInventory mirrors emir-core AgentMonitorInventory.
type AgentMonitorInventory struct {
	Manufacturer     *string `json:"manufacturer,omitempty"`
	Model            *string `json:"model,omitempty"`
	SerialNumber     *string `json:"serial_number,omitempty"`
	Tag              *string `json:"tag,omitempty"`
	Screen           *string `json:"screen,omitempty"`
	ResolutionWidth  *int    `json:"resolution_width,omitempty"`
	ResolutionHeight *int    `json:"resolution_height,omitempty"`
	RefreshRate      *float64 `json:"refresh_rate,omitempty"`
	IsPrimary        *bool   `json:"is_primary,omitempty"`
}

// AgentNetworkInterfaceInventory mirrors emir-core AgentNetworkInterfaceInventory.
type AgentNetworkInterfaceInventory struct {
	Type                   string  `json:"type"`
	MACAddress             *string `json:"mac_address,omitempty"`
	IPv4                   *string `json:"ipv4,omitempty"`
	IPv6                   *string `json:"ipv6,omitempty"`
	Speed                  *float64 `json:"speed,omitempty"`
	DNS                    *string `json:"dns,omitempty"`
	Gateway                *string `json:"gateway,omitempty"`
	RustdeskID             *string `json:"rustdesk_id,omitempty"`
	RustdeskKey            *string `json:"rustdesk_key,omitempty"`
	IsActiveDirectoryJoined *bool  `json:"is_active_directory_joined,omitempty"`
	IsCurrentlyConnected    *bool  `json:"is_currently_connected,omitempty"`
}

// AgentPeripheralInventory mirrors emir-core AgentPeripheralInventory.
type AgentPeripheralInventory struct {
	Name         string  `json:"name"`
	Model        *string `json:"model,omitempty"`
	SerialNumber *string `json:"serial_number,omitempty"`
	Tag          *string `json:"tag,omitempty"`
}

// AgentInventoryRequest mirrors emir-core AgentInventoryRequest.
type AgentInventoryRequest struct {
	Computer          AgentComputerInventory             `json:"computer"`
	Disks             []AgentDiskInventory               `json:"disks,omitempty"`
	GPUs              []AgentGPUInventory                `json:"gpus,omitempty"`
	MemoryModules     []AgentMemoryModuleInventory       `json:"memory_modules,omitempty"`
	Monitors          []AgentMonitorInventory            `json:"monitors,omitempty"`
	NetworkInterfaces []AgentNetworkInterfaceInventory   `json:"network_interfaces,omitempty"`
	Peripherals       []AgentPeripheralInventory         `json:"peripherals,omitempty"`
}

// AgentPairRequest mirrors emir-core AgentPairRequest.
type AgentPairRequest struct {
	PairingCode  string  `json:"pairing_code"`
	PublicKey    string  `json:"public_key"`
	Hostname     *string `json:"hostname,omitempty"`
	SerialNumber *string `json:"serial_number,omitempty"`
	HardwareUUID *string `json:"hardware_uuid,omitempty"`
}

// AgentPairResponse mirrors emir-core AgentPairResponse.
type AgentPairResponse struct {
	ComputerIdentifier string `json:"computer_identifier"`
	AgentToken         string `json:"agent_token"`
}

// AgentHeartbeatResponse mirrors emir-core AgentHeartbeatResponse.
type AgentHeartbeatResponse struct {
	ComputerIdentifier string `json:"computer_identifier"`
	NextPollInSeconds  int    `json:"next_poll_in_seconds"`
}

// AgentVersionAsset groups the download URL and checksum for a specific platform.
type AgentVersionAsset struct {
	Platform    string `json:"platform"`
	DownloadURL string `json:"download_url"`
	Checksum    string `json:"checksum"`
}

// AgentVersionResponse mirrors emir-core AgentVersionResponse.
type AgentVersionResponse struct {
	Version            string  `json:"version"`
	DownloadURL        string  `json:"download_url"`
	Checksum           string  `json:"checksum"`
	LinuxDownloadURL   *string `json:"linux_download_url,omitempty"`
	LinuxChecksum      *string `json:"linux_checksum,omitempty"`
	MacOSDownloadURL   *string `json:"macos_download_url,omitempty"`
	MacOSChecksum      *string `json:"macos_checksum,omitempty"`
	IsMandatory        bool    `json:"is_mandatory"`
	ReleaseNotes       *string `json:"release_notes,omitempty"`
}

// AssetForPlatform returns the download URL and checksum for the given platform.
// Supported platforms: windows, linux, darwin (macOS).
func (r *AgentVersionResponse) AssetForPlatform(platform string) (downloadURL, checksum string, ok bool) {
	switch platform {
	case "windows":
		return r.DownloadURL, r.Checksum, r.DownloadURL != "" && r.Checksum != ""
	case "linux":
		if r.LinuxDownloadURL != nil && r.LinuxChecksum != nil {
			return *r.LinuxDownloadURL, *r.LinuxChecksum, true
		}
	case "darwin":
		if r.MacOSDownloadURL != nil && r.MacOSChecksum != nil {
			return *r.MacOSDownloadURL, *r.MacOSChecksum, true
		}
	}
	return "", "", false
}

// AgentState is persisted locally between runs.
type AgentState struct {
	ComputerIdentifier string `json:"computer_identifier"`
	AgentToken         string `json:"agent_token"`
	Version            string `json:"version"`
}
