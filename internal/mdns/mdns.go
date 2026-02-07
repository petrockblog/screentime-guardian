package mdns

import (
	"context"
	"log"
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

	hostname := "Screentime Guardian"

	server, err := zeroconf.Register(
		hostname,
		"_http._tcp",
		"local.",
		port,
		[]string{
			"version=1.0",
			"path=/",
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	log.Printf("mDNS: Advertising screentime-guardian.local:%d", port)

	go func() {
		<-ctx.Done()
		server.Shutdown()
	}()

	return &Service{server: server}, nil
}

// Stop stops the mDNS advertisement
func (s *Service) Stop() {
	if s.server != nil {
		s.server.Shutdown()
	}
}
