package types

// ServiceIcon represents a service icon button on the landing page
type ServiceIcon struct {
	Name    string
	Label   string
	Href    string
	IconSVG string
	Enabled bool
}

// GetDefaultServiceIcons returns the default list of service icons
func GetDefaultServiceIcons() []ServiceIcon {
	return []ServiceIcon{
		{
			Name:    "cirrus-drive",
			Label:   "Cirrus Drive",
			Href:    "/files",
			Enabled: true,
			IconSVG: `<path d="M3 15v4c0 1.1.9 2 2 2h14a2 2 0 0 0 2-2v-4M17 8l-5-5-5 5M12 3v12"/>`,
		},
		{
			Name:    "photos",
			Label:   "Photos",
			Href:    "/photos",
			Enabled: true,
			IconSVG: `<rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
					<circle cx="8.5" cy="8.5" r="1.5"/>
					<path d="m21 15-5-5L5 21"/>`,
		},
		{
			Name:    "calendar",
			Label:   "Calendar",
			Href:    "/calendar",
			Enabled: true,
			IconSVG: `<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
				<line x1="16" y1="2" x2="16" y2="6"/>
				<line x1="8" y1="2" x2="8" y2="6"/>
				<line x1="3" y1="10" x2="21" y2="10"/>`,
		},
		{
			Name:    "books",
			Label:   "Books",
			Href:    "/books",
			Enabled: true,
			IconSVG: `<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
				<path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>`,
		},
		{
			Name:    "docs",
			Label:   "Docs",
			Href:    "/files",
			Enabled: false,
			IconSVG: `<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
				<polyline points="14 2 14 8 20 8"/>
				<line x1="16" y1="13" x2="8" y2="13"/>
				<line x1="16" y1="17" x2="8" y2="17"/>
				<polyline points="10 9 9 9 8 9"/>`,
		},
		{
			Name:    "passwords",
			Label:   "Passwords",
			Href:    "#",
			Enabled: false,
			IconSVG: `<rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
					<path d="M7 11V7a5 5 0 0 1 10 0v4"/>`,
		},
		{
			Name:    "vpn",
			Label:   "VPN",
			Href:    "#",
			Enabled: false,
			IconSVG: `<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>`,
		},
		{
			Name:    "email",
			Label:   "Email",
			Href:    "#",
			Enabled: false,
			IconSVG: `<path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
				<polyline points="22,6 12,13 2,6"/>`,
		},
		{
			Name:    "backups",
			Label:   "Backups",
			Href:    "#",
			Enabled: false,
			IconSVG: `<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
					<polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
					<line x1="12" y1="22.08" x2="12" y2="12"/>`,
		},
	}
}
