package containerclient_test

import (
	"context"
	"testing"

	"github.com/dcm-project/3-tier-demo-service-provider/api/v1alpha1"
	"github.com/dcm-project/3-tier-demo-service-provider/internal/containerclient"
)

func TestHTTPClient_AgainstMockServer(t *testing.T) {
	srv := containerclient.MockContainerServer()
	defer srv.Close()

	client, err := containerclient.NewHTTPClient(srv.URL)
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	ctx := context.Background()
	spec := v1alpha1.ThreeTierSpec{
		Database: v1alpha1.DatabaseTierSpec{Engine: "postgres", Version: "15"},
		App:      v1alpha1.AppTierSpec{Image: "spring-petclinic:latest"},
		Web:      v1alpha1.WebTierSpec{Image: "nginx:alpine"},
	}

	dbID, appID, webID, err := client.CreateContainers(ctx, "stack1", spec)
	if err != nil {
		t.Fatalf("CreateContainers: %v", err)
	}
	if dbID != "stack1-db" || appID != "stack1-app" || webID != "stack1-web" {
		t.Errorf("got ids %q %q %q", dbID, appID, webID)
	}

	_, _, _, err = client.CreateContainers(ctx, "stack1", spec)
	if err != containerclient.ErrConflict {
		t.Errorf("duplicate create: want ErrConflict, got %v", err)
	}

	if err := client.DeleteContainers(ctx, "stack1"); err != nil {
		t.Errorf("DeleteContainers: %v", err)
	}

	if err := client.DeleteContainers(ctx, "stack1"); err != containerclient.ErrNotFound {
		t.Errorf("delete non-existent: want ErrNotFound, got %v", err)
	}
}

func TestHTTPClient_TierSpecPortsMapped(t *testing.T) {
	srv := containerclient.MockContainerServer()
	defer srv.Close()

	client, err := containerclient.NewHTTPClient(srv.URL)
	if err != nil {
		t.Fatalf("NewHTTPClient: %v", err)
	}
	ctx := context.Background()
	dbPorts := []v1alpha1.ContainerPort{{ContainerPort: 5433}}
	appPorts := []v1alpha1.ContainerPort{{ContainerPort: 9090}, {ContainerPort: 9091}}
	spec := v1alpha1.ThreeTierSpec{
		Database: v1alpha1.DatabaseTierSpec{
			Engine:  "postgres",
			Version: "15",
			Network: &v1alpha1.TierNetwork{Ports: &dbPorts},
		},
		App: v1alpha1.AppTierSpec{
			Image:   "app:latest",
			Network: &v1alpha1.TierNetwork{Ports: &appPorts},
		},
		Web: v1alpha1.WebTierSpec{Image: "nginx:alpine"}, // no Network -> default 80
	}

	_, _, _, err = client.CreateContainers(ctx, "stack2", spec)
	if err != nil {
		t.Fatalf("CreateContainers: %v", err)
	}
	// Mock server accepts; we only verify no error (ports passed in request body)
}
