package odata

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func handleGetEntity(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling GET request for entitySet: %s", chi.URLParam(r, "entitySet"))
	entitySet := chi.URLParam(r, "entitySet")
	handler, ok := entityHandlers[entitySet]
	if !ok || handler.GetEntityHandler == nil {
		http.Error(w, "Entity set not found or GetEntityHandler not implemented", http.StatusNotFound)
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
	if !ok || handler.GetEntityByIDHandler == nil {
		http.Error(w, "Entity set not found or GetEntityByIDHandler not implemented", http.StatusNotFound)
		return
	}
	handler.GetEntityByIDHandler(w, r, id)
}