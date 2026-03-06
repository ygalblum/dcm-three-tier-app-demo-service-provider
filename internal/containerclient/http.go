package containerclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dcm-project/3-tier-demo-service-provider/api/v1alpha1"
	k8sapi "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
	k8sclient "github.com/dcm-project/k8s-container-service-provider/pkg/client"
)

// HTTPClient calls the k8s container SP over HTTP. Uses Container IDs
// stackID-db, stackID-app, stackID-web for idempotent creation.
type HTTPClient struct {
	Client *k8sclient.ClientWithResponses
}

// NewHTTPClient creates an HTTP client for the given base URL.
func NewHTTPClient(baseURL string) (*HTTPClient, error) {
	client, err := k8sclient.NewClientWithResponses(baseURL)
	if err != nil {
		return nil, err
	}
	return &HTTPClient{Client: client}, nil
}

func tierPorts(net *v1alpha1.TierNetwork, defaultPort int) []k8sapi.ContainerPort {
	if net != nil && net.Ports != nil && len(*net.Ports) > 0 {
		out := make([]k8sapi.ContainerPort, len(*net.Ports))
		for i, p := range *net.Ports {
			out[i] = k8sapi.ContainerPort{ContainerPort: p.ContainerPort}
		}
		return out
	}
	return []k8sapi.ContainerPort{{ContainerPort: defaultPort}}
}

func (h *HTTPClient) CreateContainers(ctx context.Context, stackID string, spec v1alpha1.ThreeTierSpec) (dbID, appID, webID string, err error) {
	tiers := []struct {
		name   string
		id     string
		image  string
		ports  []k8sapi.ContainerPort
	}{
		{name: "db", id: stackID + "-db", image: dbImageFromSpec(spec.Database), ports: tierPorts(spec.Database.Network, 5432)},
		{name: "app", id: stackID + "-app", image: spec.App.Image, ports: tierPorts(spec.App.Network, 8080)},
		{name: "web", id: stackID + "-web", image: spec.Web.Image, ports: tierPorts(spec.Web.Network, 80)},
	}

	ids := make([]string, 0, 3)
	for _, t := range tiers {
		body := k8sapi.Container{
			ServiceType: k8sapi.ContainerServiceTypeContainer,
			Metadata: k8sapi.ContainerMetadata{Name: t.id},
			Image:    k8sapi.ContainerImage{Reference: t.image},
			Resources: k8sapi.ContainerResources{
				Cpu:    k8sapi.ContainerCpu{Min: 1, Max: 2},
				Memory: k8sapi.ContainerMemory{Min: "256MB", Max: "512MB"},
			},
			Network: &k8sapi.ContainerNetwork{Ports: t.ports},
		}
		idParam := t.id
		resp, err := h.Client.CreateContainerWithResponse(ctx, &k8sapi.CreateContainerParams{Id: &idParam}, body)
		if err != nil {
			return "", "", "", fmt.Errorf("create %s: %w", t.name, err)
		}
		switch resp.StatusCode() {
		case http.StatusCreated:
			if resp.JSON201 != nil && resp.JSON201.Id != nil {
				ids = append(ids, *resp.JSON201.Id)
			} else {
				ids = append(ids, t.id)
			}
		case http.StatusConflict:
			return "", "", "", ErrConflict
		default:
			return "", "", "", fmt.Errorf("create %s: unexpected status %d", t.name, resp.StatusCode())
		}
	}
	return ids[0], ids[1], ids[2], nil
}

func (h *HTTPClient) DeleteContainers(ctx context.Context, stackID string) error {
	ids := []string{stackID + "-db", stackID + "-app", stackID + "-web"}
	for _, id := range ids {
		resp, err := h.Client.DeleteContainerWithResponse(ctx, id)
		if err != nil {
			return fmt.Errorf("delete %s: %w", id, err)
		}
		if resp.StatusCode() == http.StatusNotFound {
			return ErrNotFound
		}
		if resp.StatusCode() != http.StatusNoContent && resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("delete %s: unexpected status %d", id, resp.StatusCode())
		}
	}
	return nil
}

// GetStatus cannot determine status from k8s container SP; caller should use stored status.
func (h *HTTPClient) GetStatus(ctx context.Context, stackID string) (v1alpha1.StackStatus, bool) {
	return "", false
}

// Ensure HTTPClient implements ContainerClient.
var _ ContainerClient = (*HTTPClient)(nil)
