package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/dcm-project/3-tier-demo-service-provider/internal/api/server"
	"github.com/dcm-project/3-tier-demo-service-provider/internal/config"
	"github.com/dcm-project/3-tier-demo-service-provider/internal/containerclient"
	"github.com/dcm-project/3-tier-demo-service-provider/internal/handlers"
	"github.com/dcm-project/3-tier-demo-service-provider/internal/registration"
	"github.com/dcm-project/3-tier-demo-service-provider/internal/statusreport"
	"github.com/dcm-project/3-tier-demo-service-provider/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	var containerClient containerclient.ContainerClient
	switch cfg.DevContainerBackend {
	case "podman":
		containerClient = &containerclient.PodmanClient{}
		log.Println("Using Podman backend (real containers)")
	case "":
		if cfg.ContainerSPURL != "" {
			httpClient, err := containerclient.NewHTTPClient(cfg.ContainerSPURL)
			if err != nil {
				logger.Error("failed to create container SP HTTP client", "error", err, "url", cfg.ContainerSPURL)
				os.Exit(1)
			}
			containerClient = httpClient
			log.Printf("Using k8s container SP at %s", cfg.ContainerSPURL)
		} else {
			containerClient = &containerclient.MockClient{}
		}
	default:
		containerClient = &containerclient.MockClient{}
	}

	if cfg.RegistrationEnabled() {
		registrar, err := registration.NewRegistrar(&cfg, logger)
		if err != nil {
			logger.Error("failed to create registrar", "error", err)
			os.Exit(1)
		}
		registrar.Start(context.Background())
		logger.Info("DCM registration started in background")
	}

	var statusReporter handlers.StatusReporter
	if cfg.DCM.StatusReportURL != "" {
		pub, err := statusreport.NewPublisher(cfg.DCM.StatusReportURL, cfg.Provider.Name)
		if err != nil {
			logger.Error("failed to create status publisher", "error", err)
			os.Exit(1)
		}
		statusReporter = pub
		log.Printf("Status reporting enabled: %s", cfg.DCM.StatusReportURL)
	}

	h := &handlers.Handlers{
		Store:     store.NewMemoryStore(),
		Container: containerClient,
		Status:    statusReporter,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	_ = server.HandlerFromMuxWithBaseURL(h, r, "/api/v1alpha1")
	log.Printf("Listening on %s", cfg.SVCAddress)
	log.Fatal(http.ListenAndServe(cfg.SVCAddress, r))
}
