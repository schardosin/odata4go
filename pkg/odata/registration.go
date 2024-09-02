package odata

import (
	"log"

	"github.com/go-chi/chi/v5"
)

func RegisterEntity(entity Entity, handler EntityHandler) {
	entityName := entity.EntityName()
	entityHandlers[entityName] = handler
	entityTypes = append(entityTypes, entity)
	log.Printf("Registered entity: %s", entityName)
}

func RegisterEntityRelationship(entityName, relationshipName, targetEntityName, relationType string) {
	if entityRelationships[entityName] == nil {
		entityRelationships[entityName] = make(map[string]RelationshipInfo)
	}
	entityRelationships[entityName][relationshipName] = RelationshipInfo{
		TargetEntity: targetEntityName,
		Type:         relationType,
	}
	log.Printf("Registered relationship: %s.%s -> %s (%s)", entityName, relationshipName, targetEntityName, relationType)
}

func RegisterRoutes(router *chi.Mux) {
	router.Get("/odata/v4/$metadata", handleGetMetadata)
	router.Get("/odata/v4/{entitySet}", handleGetEntity)
	router.Get("/odata/v4/{entitySet}({id})", handleGetEntityByID)
	router.Get("/odata/v4/{entitySet}/{id}", handleGetEntityByID)
	log.Println("Registered OData routes")
}