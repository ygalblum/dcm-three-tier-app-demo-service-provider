package config

import (
	"os"
	"strings"
)

// Config holds service configuration.
type Config struct {
	// ContainerSPURL is the base URL of the container SP. Empty = use mock or podman.
	ContainerSPURL string
	// DevContainerBackend selects backend when CONTAINER_SP_URL is empty: "podman" = real containers, "mock" or empty = in-memory mock.
	DevContainerBackend string
	// SVCAddress is the listen address (e.g. ":8080"). Default ":8080".
	SVCAddress string
	// DCM holds DCM registration settings. When RegistrationURL is set, SP self-registers on startup.
	DCM DCMConfig
	// Provider holds SP identity for registration.
	Provider ProviderConfig
}

// DCMConfig holds DCM registry connection settings.
type DCMConfig struct {
	RegistrationURL string
	// StatusReportURL is the URL to POST CloudEvents for status updates (per service-provider-status-reporting).
	// When set, the SP publishes status changes to DCM.
	StatusReportURL string
}

// ProviderConfig holds service provider identity for registration.
type ProviderConfig struct {
	Name        string
	DisplayName string
	Endpoint    string
	Region      string
	Zone        string
}

// Load reads config from environment.
func Load() Config {
	addr := strings.TrimSpace(os.Getenv("SVC_ADDRESS"))
	if addr == "" {
		addr = ":8080"
	}
	return Config{
		ContainerSPURL:    strings.TrimSpace(os.Getenv("CONTAINER_SP_URL")),
		DevContainerBackend: strings.TrimSpace(strings.ToLower(os.Getenv("DEV_CONTAINER_BACKEND"))),
		SVCAddress:        addr,
		DCM: DCMConfig{
			RegistrationURL:  strings.TrimSpace(os.Getenv("SP_DCM_REGISTRATION_URL")),
			StatusReportURL:  strings.TrimSpace(os.Getenv("STATUS_REPORT_URL")),
		},
		Provider: ProviderConfig{
			Name:        strings.TrimSpace(os.Getenv("SP_PROVIDER_NAME")),
			DisplayName: strings.TrimSpace(os.Getenv("SP_PROVIDER_DISPLAY_NAME")),
			Endpoint:    strings.TrimSpace(os.Getenv("SP_PROVIDER_ENDPOINT")),
			Region:      strings.TrimSpace(os.Getenv("SP_PROVIDER_REGION")),
			Zone:        strings.TrimSpace(os.Getenv("SP_PROVIDER_ZONE")),
		},
	}
}

// RegistrationEnabled returns true when all required registration config is set.
func (c Config) RegistrationEnabled() bool {
	return c.DCM.RegistrationURL != "" && c.Provider.Name != "" && c.Provider.Endpoint != ""
}
