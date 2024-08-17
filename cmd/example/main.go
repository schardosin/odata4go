package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/schardosin/odata4go/examples/basic/routes"
	"github.com/schardosin/odata4go/pkg/odata"
)

func main() {
	r := chi.NewRouter()
	routes.SetupRoutes()
	odata.RegisterRoutes(r)

	log.Println("Routes registered")
	log.Println("Server is running on http://localhost:8000")
	err := http.ListenAndServe(":8000", r)
	if err != nil {
		log.Fatal("ListenAndServe error: ", err)
	}
}
