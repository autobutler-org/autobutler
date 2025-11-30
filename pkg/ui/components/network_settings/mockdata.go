package network_settings

import "time"

// JAMES - this is mock data, please delete

type NetworkStatus struct {
	Connected     bool
	IPAddress     string
	Hostname      string
	LastConnected time.Time
}

type TailnetDevice struct {
	ID            string
	Name          string
	IPAddress     string
	OS            string
	LastSeen      time.Time
	Online        bool
	CurrentDevice bool
}

type PendingDevice struct {
	ID          string
	Name        string
	OS          string
	RequestedAt time.Time
}

type NetworkConfig struct {
	VPNEnabled      bool
	AutoConnect     bool
	ExitNode        string
	AcceptRoutes    bool
	AdvertiseRoutes string
	MagicDNSEnabled bool
	IsExitNode      bool
}

func GetMockNetworkStatus() NetworkStatus {
	return NetworkStatus{
		Connected:     true,
		IPAddress:     "100.64.0.1",
		Hostname:      "autobutler-home",
		LastConnected: time.Now().Add(-2 * time.Hour),
	}
}

func GetMockTailnetDevices() []TailnetDevice {
	return []TailnetDevice{
		{
			ID:            "1",
			Name:          "autobutler-home",
			IPAddress:     "100.64.0.1",
			OS:            "macOS",
			LastSeen:      time.Now(),
			Online:        true,
			CurrentDevice: true,
		},
		{
			ID:            "2",
			Name:          "laptop-work",
			IPAddress:     "100.64.0.2",
			OS:            "Windows",
			LastSeen:      time.Now().Add(-30 * time.Minute),
			Online:        true,
			CurrentDevice: false,
		},
		{
			ID:            "3",
			Name:          "phone-iphone",
			IPAddress:     "100.64.0.3",
			OS:            "iOS",
			LastSeen:      time.Now().Add(-5 * time.Minute),
			Online:        true,
			CurrentDevice: false,
		},
		{
			ID:            "4",
			Name:          "tablet-ipad",
			IPAddress:     "100.64.0.4",
			OS:            "iOS",
			LastSeen:      time.Now().Add(-24 * time.Hour),
			Online:        false,
			CurrentDevice: false,
		},
	}
}

func GetMockPendingDevices() []PendingDevice {
	return []PendingDevice{
		{
			ID:          "pending-1",
			Name:        "new-laptop",
			OS:          "Windows",
			RequestedAt: time.Now().Add(-5 * time.Minute),
		},
		{
			ID:          "pending-2",
			Name:        "android-phone",
			OS:          "Android",
			RequestedAt: time.Now().Add(-15 * time.Minute),
		},
	}
}

func GetMockNetworkConfig() NetworkConfig {
	return NetworkConfig{
		VPNEnabled:      true,
		AutoConnect:     true,
		ExitNode:        "",
		AcceptRoutes:    true,
		AdvertiseRoutes: "192.168.1.0/24",
		MagicDNSEnabled: true,
		IsExitNode:      true,
	}
}
