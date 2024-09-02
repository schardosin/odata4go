package odata

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func handleGetEntity(w http.ResponseWriter, r *http.Request) {
	entitySet := chi.URLParam(r, "entitySet")
	log.Printf("Handling GET request for entitySet: %s", entitySet)

	handler, ok := entityHandlers[entitySet]
	if !ok {
		http.Error(w, "Entity set not found", http.StatusNotFound)
		return
	}

	if handler.GetEntityHandler == nil {
		http.Error(w, "GetEntityHandler not implemented", http.StatusNotImplemented)
		return
	}

	handler.GetEntityHandler(w, r)
}

func handleGetEntityByID(w http.ResponseWriter, r *http.Request) {
	entitySet := chi.URLParam(r, "entitySet")
	id := chi.URLParam(r, "id")
	id = strings.Trim(id, "()") // Remove parentheses if present
	log.Printf("Handling GET request for entity: %s, ID: %s", entitySet, id)

	handler, ok := entityHandlers[entitySet]
	if !ok {
		http.Error(w, "Entity set not found", http.StatusNotFound)
		return
	}

	if handler.GetEntityByIDHandler == nil {
		http.Error(w, "GetEntityByIDHandler not implemented", http.StatusNotImplemented)
		return
	}

	handler.GetEntityByIDHandler(w, r, id)
}

func handleGetMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := GenerateMetadata()
	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(metadata))
}