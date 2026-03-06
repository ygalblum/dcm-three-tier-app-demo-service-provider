package containerclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dcm-project/3-tier-demo-service-provider/api/v1alpha1"
)

// PodmanClient creates real containers via Podman for local testing.
// Requires: podman installed and running.
type PodmanClient struct{}

func containerNames(stackID string) (db, app, web string) {
	return stackID + "-db", stackID + "-app", stackID + "-web"
}

func networkName(stackID string) string {
	return stackID + "-net"
}

// dbImageFromSpec maps engine+version to OCI image (e.g. postgres+16 -> docker.io/library/postgres:16).
func dbImageFromSpec(db v1alpha1.DatabaseTierSpec) string {
	switch db.Engine {
	case "postgres":
		return "docker.io/library/postgres:" + db.Version
	case "mysql":
		return "docker.io/library/mysql:" + db.Version
	default:
		return "docker.io/library/postgres:17"
	}
}

func (p *PodmanClient) CreateContainers(ctx context.Context, stackID string, spec v1alpha1.ThreeTierSpec) (dbID, appID, webID string, err error) {
	dbName, appName, webName := containerNames(stackID)
	netName := networkName(stackID)
	log.Printf("[podman] Creating stack %q: network=%s, db=%s, app=%s, web=%s", stackID, netName, dbName, appName, webName)

	// Create network
	if out, err := exec.CommandContext(ctx, "podman", "network", "create", netName).CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "already exists") {
			return "", "", "", fmt.Errorf("create network: %w: %s", err, string(out))
		}
	}

	// 1. DB container (Postgres)
	dbEnv := []string{"POSTGRES_PASSWORD=petclinic", "POSTGRES_DB=petclinic"}
	args := []string{"run", "-d", "--name", dbName, "--network", netName}
	for _, e := range dbEnv {
		args = append(args, "-e", e)
	}
	args = append(args, dbImageFromSpec(spec.Database))
	if out, err := exec.CommandContext(ctx, "podman", args...).CombinedOutput(); err != nil {
		return "", "", "", fmt.Errorf("create db container: %w: %s", err, string(out))
	}

	// Give Postgres time to start
	select {
	case <-ctx.Done():
		return "", "", "", ctx.Err()
	case <-time.After(3 * time.Second):
	}

	// 2. App container (Spring Pet Clinic)
	appEnv := []string{
		"SPRING_DATASOURCE_URL=jdbc:postgresql://" + dbName + ":5432/petclinic",
		"SPRING_DATASOURCE_USERNAME=postgres",
		"SPRING_DATASOURCE_PASSWORD=petclinic",
	}
	args = []string{"run", "-d", "--name", appName, "--network", netName}
	for _, e := range appEnv {
		args = append(args, "-e", e)
	}
	args = append(args, spec.App.Image)
	if out, err := exec.CommandContext(ctx, "podman", args...).CombinedOutput(); err != nil {
		_ = exec.CommandContext(context.Background(), "podman", "rm", "-f", dbName).Run()
		return "", "", "", fmt.Errorf("create app container: %w: %s", err, string(out))
	}

	// 3. Web container (nginx) - proxy to app, expose port 8080
	nginxConf := fmt.Sprintf("server { listen 80; location / { proxy_pass http://%s:8080; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; } }\n", appName)
	confDir := filepath.Join(os.TempDir(), "3tier-nginx", stackID)
	if err := os.MkdirAll(confDir, 0755); err != nil {
		_ = exec.CommandContext(context.Background(), "podman", "rm", "-f", appName, dbName).Run()
		return "", "", "", fmt.Errorf("create nginx config dir: %w", err)
	}
	confPath := filepath.Join(confDir, "default.conf")
	if err := os.WriteFile(confPath, []byte(nginxConf), 0644); err != nil {
		_ = exec.CommandContext(context.Background(), "podman", "rm", "-f", appName, dbName).Run()
		return "", "", "", fmt.Errorf("write nginx config: %w", err)
	}
	// Use 9080 to avoid conflict with SP on 8080. :z for SELinux (Fedora/RHEL).
	args = []string{"run", "-d", "--name", webName, "--network", netName, "-p", "9080:80",
		"-v", confPath + ":/etc/nginx/conf.d/default.conf:ro,z", spec.Web.Image}
	if out, err := exec.CommandContext(ctx, "podman", args...).CombinedOutput(); err != nil {
		_ = exec.CommandContext(context.Background(), "podman", "rm", "-f", appName, dbName).Run()
		return "", "", "", fmt.Errorf("create web container: %w: %s", err, string(out))
	}

	log.Printf("[podman] Created containers: %s, %s, %s", dbName, appName, webName)
	return dbName, appName, webName, nil
}

// GetStatus returns the worst status among the 3 containers (FAILED > PENDING > RUNNING).
// Uses podman inspect to check each container's state.
func (p *PodmanClient) GetStatus(ctx context.Context, stackID string) (v1alpha1.StackStatus, bool) {
	dbName, appName, webName := containerNames(stackID)
	states := make([]string, 0, 3)
	for _, name := range []string{dbName, appName, webName} {
		out, err := exec.CommandContext(ctx, "podman", "inspect", "-f", "{{.State.Status}}", name).CombinedOutput()
		if err != nil {
			return v1alpha1.FAILED, true
		}
		states = append(states, strings.TrimSpace(string(out)))
	}
	return WorstStatusFromPodmanStates(states)
}

func (p *PodmanClient) DeleteContainers(ctx context.Context, stackID string) error {
	dbName, appName, webName := containerNames(stackID)
	netName := networkName(stackID)

	// Stop and remove in reverse order
	for _, name := range []string{webName, appName, dbName} {
		_ = exec.CommandContext(ctx, "podman", "rm", "-f", name).Run()
	}
	_ = exec.CommandContext(ctx, "podman", "network", "rm", netName).Run()
	_ = os.RemoveAll(filepath.Join(os.TempDir(), "3tier-nginx", stackID))
	return nil
}
