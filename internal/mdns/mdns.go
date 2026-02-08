package mdns

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
)

// Service handles mDNS service advertisement
type Service struct {
	server *zeroconf.Server
}

// Start begins advertising the screentime guardian service via mDNS
// hostname: the mDNS hostname to advertise (without .local suffix)
//
//	if empty, uses "screentime-guardian-{system-hostname}"
func Start(ctx context.Context, listenAddr string, hostname string) (*Service, error) {
	port := 8080
	if parts := strings.Split(listenAddr, ":"); len(parts) == 2 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			port = p
		}
	}

	// Determine hostname to use
	if hostname == "" {
		// Get system hostname
		sysHostname, err := getSystemHostname()
		if err != nil {
			log.Printf("mDNS: Could not determine system hostname: %v", err)
			sysHostname = "default"
		}
		// Create unique hostname
		hostname = "screentime-guardian-" + sysHostname
	}

	// Sanitize hostname (remove invalid characters)
	hostname = strings.ToLower(hostname)
	hostname = strings.ReplaceAll(hostname, " ", "-")

	// Get local IP addresses for the hostname
	ips, err := getLocalIPs()
	if err != nil {
		log.Printf("mDNS: Warning - could not get local IPs: %v", err)
		ips = []string{}
	}

	// Use RegisterProxy to set a custom hostname instead of using the system's hostname
	// This allows access via http://{hostname}.local:8080 regardless of the actual hostname
	server, err := zeroconf.RegisterProxy(
		"Screentime Guardian", // Service instance name
		"_http._tcp",          // Service type
		"local.",              // Domain
		port,                  // Port
		hostname,              // Hostname (creates {hostname}.local)
		ips,                   // IP addresses
		[]string{ // TXT records
			"version=1.0",
			"path=/",
		},
		nil, // Use all network interfaces
	)
	if err != nil {
		return nil, err
	}

	log.Printf("mDNS: Advertising as %s.local:%d (IPs: %v)", hostname, port, ips)

	go func() {
		<-ctx.Done()
		server.Shutdown()
	}()

	return &Service{server: server}, nil
}

// getLocalIPs returns all non-loopback IPv4 addresses of the local machine
func getLocalIPs() ([]string, error) {
	var ips []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Only include IPv4 addresses (skip IPv6 for now)
			if ip != nil && ip.To4() != nil {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips, nil
}

// getSystemHostname returns the system's hostname (short form)
func getSystemHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "machine", err
	}

	// Remove domain suffix if present (e.g., "mint-pc.local" -> "mint-pc")
	hostname = strings.Split(hostname, ".")[0]

	return hostname, nil
}

// Stop stops the mDNS advertisement
func (s *Service) Stop() {
	if s.server != nil {
		s.server.Shutdown()
	}
}
