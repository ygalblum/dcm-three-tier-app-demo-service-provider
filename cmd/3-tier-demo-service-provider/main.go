package main

import (
	"log"
	"net/http"

	"github.com/dcm-project/3-tier-demo-service-provider/internal/api/server"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	_ = server.HandlerFromMuxWithBaseURL(&server.Unimplemented{}, r, "/api/v1alpha1")
	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
