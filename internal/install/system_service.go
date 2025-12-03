package install

import (
	"runtime"
)

const (
	systemdServiceName    = "autobutler.service"
	systemdServiceContent = `[Unit]
Description=AutoButler Service
After=network.target

[Service]
ExecStart=/usr/local/bin/autobutler serve
Environment="PORT=80"
Restart=always
StandardOutput=append:/var/log/autobutler.app
StandardError=append:/var/log/autobutler.err

[Install]
WantedBy=multi-user.target`
	plistServiceName    = "ai.autobutler.plist"
	plistServiceContent = `<!-- /Library/LaunchDaemons/ -->
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.autobutler.autobutler</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/autobutler</string>
        <string>serve</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PORT</key>
        <string>80</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/autobutler.app</string>
    <key>StandardErrorPath</key>
    <string>/var/log/autobutler.err</string>
</dict>
</plist>`
)

func buildServiceFile() string {
	switch runtime.GOOS {
	case "linux":
		return systemdServiceContent
	case "darwin":
		return plistServiceContent
	default:
		panic("unsupported operating system")
	}
}
