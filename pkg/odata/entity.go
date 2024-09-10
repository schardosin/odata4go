package odata

import (
	"net/http"
)

type Entity interface {
	EntityName() string
	GetRelationships() map[string]string
}

type ExpandHandler interface {
	ExpandEntity(entity OrderedFields, relationshipName string, subQuery string) interface{}
}

type EntityHandler struct {
	GetEntityHandler     func(http.ResponseWriter, *http.Request)
	GetEntityByIDHandler func(http.ResponseWriter, *http.Request, string)
	ExpandHandler
}

// OrderedFields represents a slice of key-value pairs to maintain field order
type OrderedFields struct {
	EntityName string
	Fields     []struct {
		Key   string
		Value interface{}
	}
}

type RelationshipInfo struct {
	TargetEntity string
	Type         string // "one-to-one", "one-to-many", etc.
}

var entityTypes = []Entity{}
var entityHandlers = make(map[string]EntityHandler)
var entityRelationships = make(map[string]map[string]RelationshipInfo)

func GetEntityHandler(entityName string) (EntityHandler, bool) {
	handler, ok := entityHandlers[entityName]
	return handler, ok
}

// DefaultExpandHandler is a fallback handler that does nothing
type DefaultExpandHandler struct{}

func (h DefaultExpandHandler) ExpandEntity(entity OrderedFields, relationshipName string, subQuery string) interface{} {
	return nil
}