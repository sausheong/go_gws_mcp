// Package tooltier loads tool tier definitions from an embedded YAML file
// and resolves tier+services queries into concrete tool/service lists.
package tooltier

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed tiers.yaml
var tiersYAML []byte

// Tier name constants; these match the YAML keys.
const (
	TierCore     = "core"
	TierExtended = "extended"
	TierComplete = "complete"
)

var validTiers = []string{TierCore, TierExtended, TierComplete}

// Loader reads and resolves tier configuration.
type Loader struct {
	// services maps service name -> tier name -> tool names.
	services map[string]map[string][]string
}

// New parses the embedded tiers.yaml.
func New() (*Loader, error) {
	var raw map[string]map[string][]string
	if err := yaml.Unmarshal(tiersYAML, &raw); err != nil {
		return nil, fmt.Errorf("parse tiers.yaml: %w", err)
	}
	return &Loader{services: raw}, nil
}

// AvailableServices returns the sorted list of services defined in the YAML.
func (l *Loader) AvailableServices() []string {
	out := make([]string, 0, len(l.services))
	for s := range l.services {
		out = append(out, s)
	}
	return out
}

// ResolveToolsFromTier returns (toolNames, serviceNames) for tools at or below
// the given tier within the optional service filter. If services is nil/empty,
// all available services are considered.
func (l *Loader) ResolveToolsFromTier(tier string, services []string) ([]string, []string, error) {
	if !isValidTier(tier) {
		return nil, nil, fmt.Errorf("unknown tier %q (valid: %v)", tier, validTiers)
	}
	if len(services) == 0 {
		services = l.AvailableServices()
	}

	tierIdx := tierIndex(tier)
	seen := make(map[string]struct{})
	var toolsOut []string
	servicesOut := make(map[string]struct{})

	for _, svc := range services {
		svcTiers, ok := l.services[svc]
		if !ok {
			continue
		}
		for i := 0; i <= tierIdx; i++ {
			for _, tool := range svcTiers[validTiers[i]] {
				if _, dup := seen[tool]; dup {
					continue
				}
				seen[tool] = struct{}{}
				toolsOut = append(toolsOut, tool)
				servicesOut[svc] = struct{}{}
			}
		}
	}

	svcList := make([]string, 0, len(servicesOut))
	for s := range servicesOut {
		svcList = append(svcList, s)
	}
	return toolsOut, svcList, nil
}

func isValidTier(t string) bool {
	for _, v := range validTiers {
		if v == t {
			return true
		}
	}
	return false
}

func tierIndex(t string) int {
	for i, v := range validTiers {
		if v == t {
			return i
		}
	}
	return -1
}
