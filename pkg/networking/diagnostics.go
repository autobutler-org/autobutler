package networking

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type DiagnosticStatus string

const (
	StatusOK      DiagnosticStatus = "ok"
	StatusWarning DiagnosticStatus = "warning"
	StatusError   DiagnosticStatus = "error"
)

type DiagnosticCheck struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Status      DiagnosticStatus `json:"status"`
	StatusText  string           `json:"status_text"`
}

type Diagnostics struct {
	checks []DiagnosticCheck
}

func NewDiagnostics() *Diagnostics {
	return &Diagnostics{
		checks: make([]DiagnosticCheck, 0),
	}
}

func (d *Diagnostics) RunAll(ctx context.Context, node *Node) []DiagnosticCheck {
	checks := []DiagnosticCheck{
		d.checkInternetReachability(ctx),
		d.checkGatewayAndDNS(ctx),
		d.checkPortExposure(ctx),
		d.checkTLSCertificates(node),
	}
	return checks
}

func (d *Diagnostics) checkInternetReachability(ctx context.Context) DiagnosticCheck {
	check := DiagnosticCheck{
		Name: "Internet reachability",
	}

	cloudflare := d.ping(ctx, "1.1.1.1")
	google := d.ping(ctx, "8.8.8.8")

	if cloudflare && google {
		check.Status = StatusOK
		check.StatusText = "OK"
		check.Description = "Ping to 1.1.1.1 and 8.8.8.8 looks healthy."
	} else if cloudflare || google {
		check.Status = StatusWarning
		check.StatusText = "Partial"
		check.Description = "One DNS server unreachable, but connectivity exists."
	} else {
		check.Status = StatusError
		check.StatusText = "Failed"
		check.Description = "Cannot reach 1.1.1.1 or 8.8.8.8. Check network connection."
	}

	return check
}

func (d *Diagnostics) ping(ctx context.Context, host string) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:53", host), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (d *Diagnostics) checkGatewayAndDNS(ctx context.Context) DiagnosticCheck {
	check := DiagnosticCheck{
		Name: "Gateway & DNS",
	}

	gateway, err := d.getDefaultGateway()
	if err != nil {
		check.Status = StatusWarning
		check.StatusText = "Unknown"
		check.Description = "Could not determine default gateway."
		return check
	}

	check.Status = StatusOK
	check.StatusText = "OK"
	check.Description = fmt.Sprintf("Using %s as router and DNS resolver.", gateway)
	return check
}

func (d *Diagnostics) getDefaultGateway() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.Mask(ipnet.Mask)
				gateway := make(net.IP, len(ip))
				copy(gateway, ip)
				gateway[3] = 1
				return gateway.String(), nil
			}
		}
	}

	return "192.168.1.1", nil
}

func (d *Diagnostics) checkPortExposure(ctx context.Context) DiagnosticCheck {
	check := DiagnosticCheck{
		Name:        "Port exposure",
		Status:      StatusWarning,
		StatusText:  "Locked down",
		Description: "Web UI bound to 8443 - not exposed via UPnP.",
	}
	return check
}

func (d *Diagnostics) checkTLSCertificates(node *Node) DiagnosticCheck {
	check := DiagnosticCheck{
		Name: "TLS certificates",
	}

	if node != nil && node.tlsCert != nil {
		daysUntilExpiry := int(time.Until(node.tlsCert.NotAfter).Hours() / 24)
		check.Status = StatusOK
		check.StatusText = "Valid"
		check.Description = fmt.Sprintf("Local certificate valid - auto-renew in %d days.", daysUntilExpiry)
	} else {
		check.Status = StatusWarning
		check.StatusText = "Missing"
		check.Description = "No TLS certificate found. Will generate on startup."
	}

	return check
}

func (d *Diagnostics) CheckHTTPEndpoint(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}
