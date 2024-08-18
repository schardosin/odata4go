package odata

import (
	"encoding/json"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
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
	ExpandEntity(entity interface{}, relationshipName string) interface{}
}

// OrderedFields represents a slice of key-value pairs to maintain field order
type OrderedFields []struct {
	Key   string
	Value interface{}
}

var entityTypes = []Entity{}
var entityHandlers = make(map[string]EntityHandler)
var entityRelationships = make(map[string]map[string]RelationshipInfo)

type RelationshipInfo struct {
	TargetEntity string
	Type         string // "one-to-one", "one-to-many", etc.
}

func RegisterEntity(entity Entity, handler EntityHandler) {
	entityTypes = append(entityTypes, entity)
	entitySetName := entity.EntityName()
	entityHandlers[entitySetName] = handler
	log.Printf("Registered entity: %s", entitySetName)
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
	router.Get("/odata/v4/{entitySet}/{id}", handleGetEntityByID) // New route for single item with "/" support
	log.Println("Registered OData routes")
}

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

func ApplySkipTop(entities interface{}, skip, top string) interface{} {
	slice := reflect.ValueOf(entities)
	if slice.Kind() != reflect.Slice {
		return entities
	}

	skipInt, _ := strconv.Atoi(skip)
	topInt, _ := strconv.Atoi(top)

	if skipInt < 0 {
		skipInt = 0
	}

	if topInt <= 0 || topInt > slice.Len()-skipInt {
		topInt = slice.Len() - skipInt
	}

	if skipInt >= slice.Len() {
		return reflect.MakeSlice(slice.Type(), 0, 0).Interface()
	}

	result := slice.Slice(skipInt, skipInt+topInt)
	return result.Interface()
}

func ApplyExpand(entities interface{}, expand string, handler ExpandHandler) interface{} {
	if expand == "" || handler == nil {
		return entities
	}

	expandedEntities := reflect.ValueOf(entities)
	if expandedEntities.Kind() == reflect.Slice {
		result := make([]OrderedFields, 0, expandedEntities.Len())

		for i := 0; i < expandedEntities.Len(); i++ {
			entity := expandedEntities.Index(i).Interface()
			expandedEntity := ApplyExpandSingle(entity, expand, handler)
			result = append(result, expandedEntity)
		}

		return result
	} else {
		return ApplyExpandSingle(entities, expand, handler)
	}
}

func ApplyExpandSingle(entity interface{}, expand string, handler ExpandHandler) OrderedFields {
	log.Printf("ApplyExpandSingle called with expand: %s", expand)
	result := EntityToOrderedFields(entity, expand)
	if expand == "" {
		return result
	}

	expandParts := parseExpandQuery(expand)
	log.Printf("Parsed expand parts: %+v", expandParts)
	
	for relationshipName, nestedExpand := range expandParts {
		log.Printf("Processing relationship: %s with nested expand: %s", relationshipName, nestedExpand)
		expandedEntity := handler.ExpandEntity(entity, relationshipName)
		if expandedEntity != nil {
			log.Printf("Expanded entity for %s: %+v", relationshipName, expandedEntity)
			// Remove the existing field if it exists
			for i, field := range result {
				if field.Key == relationshipName {
					result = append(result[:i], result[i+1:]...)
					break
				}
			}

			// Check if the expanded entity is a slice (one-to-many relationship)
			expandedValue := reflect.ValueOf(expandedEntity)
			if expandedValue.Kind() == reflect.Slice {
				log.Printf("Expanded entity is a slice with %d elements", expandedValue.Len())
				// Convert each item in the slice to OrderedFields
				expandedSlice := make([]OrderedFields, expandedValue.Len())
				for i := 0; i < expandedValue.Len(); i++ {
					expandedSlice[i] = ApplyExpandSingle(expandedValue.Index(i).Interface(), nestedExpand, handler)
				}
				// Add the expanded result
				result = append(result, struct{Key string; Value interface{}}{relationshipName, expandedSlice})
			} else {
				log.Printf("Expanded entity is a single entity")
				// Handle one-to-one relationship
				expandedOrderedFields := ApplyExpandSingle(expandedEntity, nestedExpand, handler)
				result = append(result, struct{Key string; Value interface{}}{relationshipName, expandedOrderedFields})
			}
		} else {
			log.Printf("ExpandEntity returned nil for %s", relationshipName)
		}
	}

	log.Printf("Final result: %+v", result)
	return result
}

func parseExpandQuery(expand string) map[string]string {
	expandParts := make(map[string]string)
	currentKey := ""
	nestedLevel := 0
	var currentValue strings.Builder

	for _, char := range expand {
		switch char {
		case '(':
			if nestedLevel == 0 {
				currentValue.WriteRune(char)
			}
			nestedLevel++
		case ')':
			nestedLevel--
			if nestedLevel == 0 {
				currentValue.WriteRune(char)
				expandParts[currentKey] = strings.TrimPrefix(strings.TrimSuffix(currentValue.String(), ")"), "($expand=")
				currentKey = ""
				currentValue.Reset()
			} else {
				currentValue.WriteRune(char)
			}
		case ',':
			if nestedLevel == 0 {
				if currentKey != "" {
					expandParts[currentKey] = currentValue.String()
				}
				currentKey = ""
				currentValue.Reset()
			} else {
				currentValue.WriteRune(char)
			}
		default:
			if nestedLevel == 0 {
				if currentValue.Len() == 0 {
					currentKey += string(char)
				} else {
					currentValue.WriteRune(char)
				}
			} else {
				currentValue.WriteRune(char)
			}
		}
	}

	if currentKey != "" {
		expandParts[currentKey] = currentValue.String()
	}

	return expandParts
}

func ApplySelect(entities interface{}, selectQuery string) interface{} {
	if selectQuery == "" {
		return entities
	}

	log.Printf("ApplySelect: Input entities: %+v", entities)
	log.Printf("ApplySelect: Select query: %s", selectQuery)

	selectedFields := strings.Split(selectQuery, ",")
	for i := range selectedFields {
		selectedFields[i] = strings.TrimSpace(selectedFields[i])
	}

	log.Printf("ApplySelect: Selected fields: %v", selectedFields)

	entitiesValue := reflect.ValueOf(entities)
	if entitiesValue.Kind() == reflect.Slice {
		result := make([]OrderedFields, 0, entitiesValue.Len())
		for i := 0; i < entitiesValue.Len(); i++ {
			entity := entitiesValue.Index(i).Interface()
			selectedEntity := ApplySelectSingle(EntityToOrderedFields(entity, ""), selectedFields)
			result = append(result, selectedEntity)
		}
		log.Printf("ApplySelect: Result: %+v", result)
		return result
	} else {
		result := ApplySelectSingle(EntityToOrderedFields(entities, ""), selectedFields)
		log.Printf("ApplySelect: Result for single entity: %+v", result)
		return result
	}
}

func ApplySelectSingle(entity OrderedFields, selectedFields []string) OrderedFields {
	log.Printf("ApplySelectSingle: Input entity: %+v", entity)
	log.Printf("ApplySelectSingle: Selected fields: %v", selectedFields)

	if len(selectedFields) == 0 {
		return entity
	}

	result := make(OrderedFields, 0, len(selectedFields))

	for _, field := range selectedFields {
		for _, kv := range entity {
			if strings.EqualFold(kv.Key, field) {
				result = append(result, kv)
				break
			}
		}
	}

	log.Printf("ApplySelectSingle: Result: %+v", result)
	return result
}

func EntityToOrderedFields(entity interface{}, expand string) OrderedFields {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	result := make(OrderedFields, 0, val.NumField())

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)
		
		// Check if the field is expandable
		odataTag := field.Tag.Get("odata")
		if strings.HasPrefix(odataTag, "expand:") {
			expandField := strings.TrimPrefix(odataTag, "expand:")
			if expand == "" || !containsField(expand, expandField) {
				continue // Skip this field if it's not expanded
			}
		}
		
		result = append(result, struct{Key string; Value interface{}}{field.Name, fieldValue.Interface()})
	}

	return result
}

// Helper function to check if a field is in the expand query
func containsField(expand, field string) bool {
	fields := strings.Split(expand, ",")
	for _, f := range fields {
		if strings.TrimSpace(f) == field {
			return true
		}
	}
	return false
}

// Helper function to create OData response for multiple entities
func CreateODataResponse(w http.ResponseWriter, entitySet string, entities interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("OData-Version", "4.0")

	var orderedEntities interface{}
	entitiesValue := reflect.ValueOf(entities)
	
	if entitiesValue.Kind() == reflect.Slice {
		orderedSlice := make([]interface{}, 0, entitiesValue.Len())
		for i := 0; i < entitiesValue.Len(); i++ {
			entity := entitiesValue.Index(i).Interface()
			switch v := entity.(type) {
			case OrderedFields:
				orderedSlice = append(orderedSlice, v)
			default:
				orderedEntity := EntityToOrderedFields(entity, "")
				orderedSlice = append(orderedSlice, orderedEntity)
			}
		}
		orderedEntities = orderedSlice
	} else {
		switch v := entities.(type) {
		case OrderedFields:
			orderedEntities = v
		default:
			orderedEntities = EntityToOrderedFields(entities, "")
		}
	}

	response := OrderedFields{
		{Key: "@odata.context", Value: "$metadata#" + entitySet},
		{Key: "value", Value: orderedEntities},
	}
	encodeJSONPreserveOrder(w, response)
}

// Helper function to create OData response for a single entity
func CreateODataResponseSingle(w http.ResponseWriter, entitySet string, entity interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("OData-Version", "4.0")
	
	var response OrderedFields
	response = append(response, struct{Key string; Value interface{}}{"@odata.context", "$metadata#" + entitySet + "/$entity"})
	
	switch v := entity.(type) {
	case OrderedFields:
		response = append(response, v...)
	default:
		orderedEntity := EntityToOrderedFields(entity, "")
		response = append(response, orderedEntity...)
	}
	
	encodeJSONPreserveOrder(w, response)
}

// Helper function to encode JSON while preserving field order
func encodeJSONPreserveOrder(w http.ResponseWriter, v interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(v)
}

func (of OrderedFields) MarshalJSON() ([]byte, error) {
	var buf strings.Builder
	buf.WriteString("{")
	for i, kv := range of {
		if i > 0 {
			buf.WriteString(",")
		}
		// Marshal the key
		key, err := json.Marshal(kv.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteString(":")
		// Marshal the value
		val, err := json.Marshal(kv.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteString("}")
	return []byte(buf.String()), nil
}
