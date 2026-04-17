package server

import (
	"context"
	"log"
	"net"
	"strings"

	"laps/config"

	"github.com/hashicorp/mdns"
)

// StartMDNS publishes the mDNS service name "laps.local" for the given
// configuration. It returns a stop function which should be called to shut
// down the mdns server, or the returned server will be shut down automatically
// when the provided context is cancelled.
func StartMDNS(ctx context.Context, cfg *config.Config) (func(), error) {
	// Build IP list for the host (exclude loopback)
	ips, err := getLocalIPs()
	if err != nil {
		return nil, err
	}

	// Ensure port is set
	port := cfg.ListenPort
	if port == 0 {
		port = 8080
	}

	instance := "laps"
	service := "_http._tcp."
	domain := "local."
	// hostName must be a fully-qualified domain name ending with a dot
	hostName := "laps.local."

	svc, err := mdns.NewMDNSService(instance, service, domain, hostName, port, ips, nil)
	if err != nil {
		return nil, err
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: svc})
	if err != nil {
		return nil, err
	}

	// Shutdown on context cancel
	go func() {
		<-ctx.Done()
		log.Println("mdns: context done, shutting down mdns server")
		server.Shutdown()
	}()

	stop := func() {
		_ = server.Shutdown()
	}

	return stop, nil
}

// getLocalIPs returns non-loopback IP addresses for the host.
func getLocalIPs() ([]net.IP, error) {
	var ips []net.IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, intf := range ifaces {
		// skip down or loopback interfaces
		if intf.Flags&net.FlagUp == 0 || intf.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := intf.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			// skip link-local multicast, unspecified, etc
			if ip.IsLoopback() || ip.IsUnspecified() {
				continue
			}
			// normalize IPv4-mapped IPv6
			if ip4 := ip.To4(); ip4 != nil {
				ip = ip4
			}
			ips = append(ips, ip)
		}
	}

	// dedupe
	uniq := make(map[string]net.IP)
	for _, ip := range ips {
		uniq[ip.String()] = ip
	}
	out := make([]net.IP, 0, len(uniq))
	for _, ip := range uniq {
		// avoid link-local addresses that may not be useful
		if strings.HasPrefix(ip.String(), "fe80:") {
			continue
		}
		out = append(out, ip)
	}
	return out, nil
}
