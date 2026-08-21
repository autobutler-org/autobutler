package install

import (
	"runtime"
)

const (
	serviceUserName    = "quark"
	serviceDataDir     = "/var/lib/quark"
	systemdServiceName = "quark.service"

	systemdServiceContent = `[Unit]
Description=Quark Service
After=network.target

[Service]
User=quark
Group=quark
ExecStart=/usr/local/bin/quark serve
Environment="PORT=80"
Environment="HTTPS_PORT=443"
Environment="GIN_MODE=release"
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
Restart=always
StandardOutput=append:/var/log/quark.app
StandardError=append:/var/log/quark.err

[Install]
WantedBy=multi-user.target`
	plistServiceName    = "ai.quark.plist"
	plistServiceContent = `<!-- /Library/LaunchDaemons/ -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>org.quark</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/quark</string>
        <string>serve</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PORT</key>
        <string>80</string>
        <key>HTTPS_PORT</key>
        <string>443</string>
        <key>GIN_MODE</key>
        <string>release</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/quark.app</string>
    <key>StandardErrorPath</key>
    <string>/var/log/quark.err</string>
</dict>
</plist>`
)

func buildServiceFile() string {
	switch runtime.GOOS {
	case "linux":
		return systemdServiceContent
	case "darwin": // coverage: ignore - Not run in CI
		return plistServiceContent
	default:
		panic("unsupported operating system")
	}
}
