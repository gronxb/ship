package deploy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
)

func selectComposeService(services map[string]composeService, requested string) (string, composeService, error) {
	if requested != "" {
		service, ok := services[requested]
		if !ok {
			return "", composeService{}, fmt.Errorf("Compose service %q not found", requested)
		}
		return requested, service, nil
	}
	if service, ok := services["gateway"]; ok {
		return "gateway", service, nil
	}
	if len(services) == 0 {
		return "", composeService{}, errors.New("Compose service selection failed: project has no services")
	}
	names := make([]string, 0, len(services))
	for name, service := range services {
		if len(service.Ports) > 0 {
			names = append(names, name)
		}
	}
	if len(names) == 0 && len(services) == 1 {
		for name, service := range services {
			return name, service, nil
		}
	}
	sort.Strings(names)
	if len(names) == 1 {
		return names[0], services[names[0]], nil
	}
	return "", composeService{}, fmt.Errorf("Compose service is ambiguous; use --compose-service (candidates: %s)", strings.Join(names, ", "))
}

func selectComposePort(serviceName string, ports []composePort, targetPort int) (int, error) {
	type candidate struct {
		target    int
		published int
	}
	var candidates []candidate
	seen := make(map[candidate]struct{})
	loopbackOnly := false
	ipv6Only := false
	unsupportedBinding := false
	for _, port := range ports {
		if port.Protocol != "" && !strings.EqualFold(port.Protocol, "tcp") {
			continue
		}
		published, err := parsePublishedPort(port.Published)
		if err != nil || published == 0 {
			continue
		}
		if isLoopbackBinding(port.HostIP) {
			loopbackOnly = true
			continue
		}
		if !isReachableIPv4Binding(port.HostIP) {
			if ip := net.ParseIP(port.HostIP); ip != nil && ip.To4() == nil {
				ipv6Only = true
			} else {
				unsupportedBinding = true
			}
			continue
		}
		if targetPort == 0 || port.Target == targetPort {
			value := candidate{target: port.Target, published: published}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			candidates = append(candidates, value)
		}
	}
	if len(candidates) == 0 {
		if ipv6Only {
			return 0, fmt.Errorf("Compose service %s published port is IPv6-only; Ship requires an IPv4 all-interface binding", serviceName)
		}
		if loopbackOnly {
			return 0, fmt.Errorf("Compose service %s published port is bound to loopback", serviceName)
		}
		if unsupportedBinding {
			return 0, fmt.Errorf("Compose service %s published port uses an interface-specific host IP; Ship requires 0.0.0.0", serviceName)
		}
		return 0, fmt.Errorf("Compose service %s requires a published TCP port", serviceName)
	}
	for _, candidate := range candidates {
		if candidate.target == 80 {
			return candidate.published, nil
		}
	}
	if len(candidates) == 1 {
		return candidates[0].published, nil
	}
	return 0, fmt.Errorf("Compose service %s has multiple published TCP ports; use --port to select the container port", serviceName)
}

func parsePublishedPort(raw json.RawMessage) (int, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0, nil
	}
	var text string
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &text); err != nil {
			return 0, err
		}
	} else {
		text = string(raw)
	}
	port, err := strconv.Atoi(text)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid published port %q", text)
	}
	return port, nil
}

func isLoopbackBinding(hostIP string) bool {
	if hostIP == "" {
		return false
	}
	ip := net.ParseIP(hostIP)
	return ip != nil && ip.IsLoopback()
}

func isReachableIPv4Binding(hostIP string) bool {
	if hostIP == "" {
		return true
	}
	ip := net.ParseIP(hostIP)
	return ip != nil && ip.To4() != nil && ip.IsUnspecified()
}
