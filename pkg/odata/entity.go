package odata

import (
	"net/http"
)

type Entity interface {
	EntityName() string
	GetRelationships() map[string]string
}

type EntityHandler struct {
	GetEntityHandler     func(http.ResponseWriter, *http.Request)
	GetEntityByIDHandler func(http.ResponseWriter, *http.Request, string)
	ExpandHandler        ExpandHandler
}

type ExpandHandler interface {
	ExpandEntity(entity interface{}, relationshipName string, subQuery string) interface{}
}

// OrderedFields represents a slice of key-value pairs to maintain field order
type OrderedFields []struct {
	Key   string
	Value interface{}
}

type RelationshipInfo struct {
	TargetEntity string
	Type         string // "one-to-one", "one-to-many", etc.
}

var entityTypes = []Entity{}
var entityHandlers = make(map[string]EntityHandler)
var entityRelationships = make(map[string]map[string]RelationshipInfo)