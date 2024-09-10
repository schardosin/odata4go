package odata

import (
	"encoding/json"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

func EntityToOrderedFields(entity interface{}, expand string) OrderedFields {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var entityName string
	if e, ok := entity.(Entity); ok {
		entityName = e.EntityName()
	}
	if e, ok := entity.(OrderedFields); ok {
		entityName = e.EntityName
	}

	result := OrderedFields{EntityName: entityName}

	switch val.Kind() {
	case reflect.Struct:
		typ := val.Type()
		result.Fields = make([]struct{Key string; Value interface{}}, 0, val.NumField())

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
			
			result.Fields = append(result.Fields, struct{Key string; Value interface{}}{field.Name, fieldValue.Interface()})
		}

	case reflect.Map:
		keys := val.MapKeys()
		result.Fields = make([]struct{Key string; Value interface{}}, 0, len(keys))
		
		// Sort keys to ensure consistent order
		sortedKeys := make([]string, len(keys))
		for i, key := range keys {
			sortedKeys[i] = key.String()
		}
		sort.Strings(sortedKeys)

		for _, key := range sortedKeys {
			value := val.MapIndex(reflect.ValueOf(key))
			result.Fields = append(result.Fields, struct{Key string; Value interface{}}{key, value.Interface()})
		}

	case reflect.Slice:
		if orderedFields, ok := val.Interface().(OrderedFields); ok {
			result = orderedFields
		}
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
		Fields: []struct{Key string; Value interface{}}{
			{Key: "@odata.context", Value: "$metadata#" + entitySet},
			{Key: "value", Value: orderedEntities},
		},
	}
	encodeJSONPreserveOrder(w, response)
}

// Helper function to create OData response for a single entity
func CreateODataResponseSingle(w http.ResponseWriter, entitySet string, entity interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("OData-Version", "4.0")

	// Log the type of the entity
	log.Printf("CreateODataResponseSingle: Entity type: %T", entity)
	
	var orderedEntity OrderedFields
	switch v := entity.(type) {
	case OrderedFields:
		log.Println("CreateODataResponseSingle: Entity is OrderedFields")
		orderedEntity = v
	case []OrderedFields:
		log.Println("CreateODataResponseSingle: Entity is []OrderedFields")
		if len(v) > 0 {
			orderedEntity = v[0]
		} else {
			orderedEntity = OrderedFields{Fields: []struct{Key string; Value interface{}}{}}
		}
	case map[string]interface{}:
		log.Println("CreateODataResponseSingle: Entity is map[string]interface{}")
		orderedEntity = OrderedFields{Fields: make([]struct{Key string; Value interface{}}, 0, len(v))}
		for key, value := range v {
			orderedEntity.Fields = append(orderedEntity.Fields, struct{Key string; Value interface{}}{key, value})
		}
	default:
		log.Printf("CreateODataResponseSingle: Entity is of type %T, converting to OrderedFields", v)
		orderedEntity = EntityToOrderedFields(entity, "")
	}

	// Add @odata.context to the beginning of the OrderedFields
	contextField := struct{Key string; Value interface{}}{"@odata.context", "$metadata#" + entitySet + "/$entity"}
	orderedEntity.Fields = append([]struct{Key string; Value interface{}}{contextField}, orderedEntity.Fields...)

	encodeJSONPreserveOrder(w, orderedEntity)
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
	
	for i, kv := range of.Fields {
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