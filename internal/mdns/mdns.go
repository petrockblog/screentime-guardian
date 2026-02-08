package mdns

import (
	"context"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
)

// Service handles mDNS service advertisement
type Service struct {
	server *zeroconf.Server
}

// Start begins advertising the screentime guardian service via mDNS
func Start(ctx context.Context, listenAddr string) (*Service, error) {
	port := 8080
	if parts := strings.Split(listenAddr, ":"); len(parts) == 2 {
		if p, err := strconv.Atoi(parts[1]); err == nil {
			port = p
		}
	}

	// Get local IP addresses for the hostname
	ips, err := getLocalIPs()
	if err != nil {
		log.Printf("mDNS: Warning - could not get local IPs: %v", err)
		ips = []string{}
	}

	// Use RegisterProxy to set a custom hostname instead of using the system's hostname
	// This allows access via http://screentime-guardian.local:8080 regardless of the actual hostname
	server, err := zeroconf.RegisterProxy(
		"Screentime Guardian",    // Service instance name
		"_http._tcp",             // Service type
		"local.",                 // Domain
		port,                     // Port
		"screentime-guardian",    // Hostname (creates screentime-guardian.local)
		ips,                      // IP addresses
		[]string{                 // TXT records
			"version=1.0",
			"path=/",
		},
		nil, // Use all network interfaces
	)
	if err != nil {
		return nil, err
	}

	log.Printf("mDNS: Advertising as screentime-guardian.local:%d (IPs: %v)", port, ips)

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

// Stop stops the mDNS advertisement
func (s *Service) Stop() {
	if s.server != nil {
		s.server.Shutdown()
	}
}
