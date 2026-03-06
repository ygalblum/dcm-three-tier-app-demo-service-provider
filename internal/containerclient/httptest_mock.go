package containerclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	k8sapi "github.com/dcm-project/k8s-container-service-provider/api/v1alpha1"
)

// MockContainerServer returns an httptest.Server that implements the k8s container
// API (POST /api/v1alpha1/containers, DELETE /api/v1alpha1/containers/{id}).
// TEST-ONLY: Used for contract testing in http_test.go. Not used by runtime.
// Stateful: Create returns 409 for duplicate IDs, Delete returns 404 for non-existent.
func MockContainerServer() *httptest.Server {
	mux := http.NewServeMux()
	state := &mockServerState{created: make(map[string]struct{})}

	mux.HandleFunc("/api/v1alpha1/containers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			state.handleCreate(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/api/v1alpha1/containers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			id := strings.TrimPrefix(r.URL.Path, "/api/v1alpha1/containers/")
			state.handleDelete(w, id)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	return httptest.NewServer(mux)
}

type mockServerState struct {
	mu      sync.RWMutex
	created map[string]struct{}
}

func (s *mockServerState) handleCreate(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id query param required for idempotent create", http.StatusBadRequest)
		return
	}

	var body k8sapi.Container
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.created[id]; exists {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"type": "ALREADY_EXISTS", "detail": "container already exists"})
		return
	}
	s.created[id] = struct{}{}

	now := time.Now()
	resp := k8sapi.Container{
		Id:          &id,
		Path:        ptr("containers/" + id),
		ServiceType: k8sapi.ContainerServiceTypeContainer,
		Metadata:    body.Metadata,
		Image:       body.Image,
		Resources:   body.Resources,
		Status:      ptr(k8sapi.RUNNING),
		CreateTime:  &now,
		UpdateTime:  &now,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *mockServerState) handleDelete(w http.ResponseWriter, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.created[id]; !exists {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"type": "NOT_FOUND", "detail": "container not found"})
		return
	}
	delete(s.created, id)
	w.WriteHeader(http.StatusNoContent)
}

func ptr[T any](v T) *T { return &v }
