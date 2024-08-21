package odata

import (
	"log"
	"reflect"
	"strconv"
	"strings"
)

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
	if expand != "" {
		log.Printf("ApplyExpandSingle called with expand: %s", expand)
	}

	if expand == "" || handler == nil {
		return entities
	}

	expandedEntities := reflect.ValueOf(entities)
	if expandedEntities.Kind() == reflect.Slice {
		result := make([]map[string]interface{}, 0, expandedEntities.Len())

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

func ApplyExpandSingle(entity interface{}, expand string, handler ExpandHandler) map[string]interface{} {
	result := EntityToMap(entity, expand)
	if expand == "" {
		return result
	}

	expandParts := parseExpandQuery(expand)	
	for relationshipName, nestedExpand := range expandParts {
		if nestedExpand != "" {
			log.Printf("Processing relationship: %s with nested expand: %s", relationshipName, nestedExpand)
		}
		expandedEntity := handler.ExpandEntity(entity, relationshipName)
		if expandedEntity != nil {
			// Remove the existing field if it exists
			delete(result, relationshipName)

			// Get the correct handler for the expanded entity
			expandedEntityType := reflect.TypeOf(expandedEntity)
			if expandedEntityType.Kind() == reflect.Slice {
				expandedEntityType = expandedEntityType.Elem()
			}
			expandedEntityValue := reflect.New(expandedEntityType).Elem().Interface()
			if entityWithName, ok := expandedEntityValue.(Entity); ok {
				expandedEntityName := entityWithName.EntityName()
				nestedHandler, ok := entityHandlers[expandedEntityName]
				if !ok {
					log.Printf("No handler found for entity type: %s", expandedEntityName)
					nestedHandler = EntityHandler{ExpandHandler: handler}
				}

				// Check if the expanded entity is a slice (one-to-many relationship)
				expandedValue := reflect.ValueOf(expandedEntity)
				if expandedValue.Kind() == reflect.Slice {
					log.Printf("Expanded entity is a slice with %d elements", expandedValue.Len())
					// Convert each item in the slice to map[string]interface{}
					expandedSlice := make([]map[string]interface{}, expandedValue.Len())
					for i := 0; i < expandedValue.Len(); i++ {
						expandedSlice[i] = ApplyExpandSingle(expandedValue.Index(i).Interface(), nestedExpand, nestedHandler.ExpandHandler)
					}
					// Add the expanded result
					result[relationshipName] = expandedSlice
				} else {
					// Handle one-to-one relationship
					expandedMap := ApplyExpandSingle(expandedEntity, nestedExpand, nestedHandler.ExpandHandler)
					result[relationshipName] = expandedMap
				}
			} else {
				log.Printf("Expanded entity does not implement Entity interface")
			}
		} else {
			log.Printf("ExpandEntity returned nil for %s", relationshipName)
		}
	}

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
			} else {
				currentValue.WriteRune(char)
			}
			nestedLevel++
		case ')':
			nestedLevel--
			currentValue.WriteRune(char)
			if nestedLevel == 0 {
				expandParts[currentKey] = currentValue.String()
				currentKey = ""
				currentValue.Reset()
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

	// Process nested expands
	for key, value := range expandParts {
		if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
			expandParts[key] = parseNestedExpand(value[1 : len(value)-1])
		}
	}

	return expandParts
}

func parseNestedExpand(nestedExpand string) string {
	if strings.HasPrefix(nestedExpand, "$expand=") {
		nestedExpand = strings.TrimPrefix(nestedExpand, "$expand=")
	}
	return nestedExpand
}

func ApplySelect(entities interface{}, selectQuery string) interface{} {
	if selectQuery == "" {
		return entities
	}

	selectedFields := strings.Split(selectQuery, ",")
	for i := range selectedFields {
		selectedFields[i] = strings.TrimSpace(selectedFields[i])
	}

	entitiesType := reflect.TypeOf(entities)
	if entitiesType.Kind() == reflect.Slice {
		entityType := entitiesType.Elem()
		log.Printf("ApplySelect: Processing entities of type: %s", entityType.Name())

		entitiesValue := reflect.ValueOf(entities)
		result := make([]map[string]interface{}, 0, entitiesValue.Len())
		for i := 0; i < entitiesValue.Len(); i++ {
			entity := entitiesValue.Index(i).Interface()
			selectedEntity := ApplySelectSingle(EntityToMap(entity, ""), selectedFields)
			result = append(result, selectedEntity)
		}
		return result
	} else {
		log.Printf("ApplySelect: Processing single entity of type: %s", entitiesType.Name())
		result := ApplySelectSingle(EntityToMap(entities, ""), selectedFields)
		return result
	}
}

func ApplySelectSingle(entity map[string]interface{}, selectedFields []string) map[string]interface{} {
	if len(selectedFields) == 0 {
		return entity
	}

	result := make(map[string]interface{})

	for key, value := range entity {
		if isExpandedEntity(value) {
			// Always include expanded entities
			result[key] = value
		} else {
			for _, field := range selectedFields {
				fieldParts := strings.Split(field, "/")
				if strings.EqualFold(key, fieldParts[0]) {
					if len(fieldParts) > 1 {
						// Handle nested selection
						switch v := value.(type) {
						case map[string]interface{}:
							nestedResult := ApplySelectSingle(v, []string{strings.Join(fieldParts[1:], "/")})
							result[key] = nestedResult
						case []map[string]interface{}:
							nestedSlice := make([]map[string]interface{}, len(v))
							for i, item := range v {
								nestedSlice[i] = ApplySelectSingle(item, []string{strings.Join(fieldParts[1:], "/")})
							}
							result[key] = nestedSlice
						default:
							// If it's not a map[string]interface{} or []map[string]interface{}, just add it as is
							result[key] = value
						}
					} else {
						result[key] = value
					}
					break
				}
			}
		}
	}

	return result
}

func isExpandedEntity(value interface{}) bool {
	switch value.(type) {
	case map[string]interface{}, []map[string]interface{}:
		return true
	default:
		return false
	}
}

func EntityToMap(entity interface{}, expand string) map[string]interface{} {
	// Check if the entity is already a map[string]interface{}
	if m, ok := entity.(map[string]interface{}); ok {
		return m
	}

	result := make(map[string]interface{})
	entityValue := reflect.ValueOf(entity)
	entityType := entityValue.Type()

	// Handle pointer types
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
		entityType = entityValue.Type()
	}

	// Handle struct types
	if entityValue.Kind() == reflect.Struct {
		for i := 0; i < entityValue.NumField(); i++ {
			field := entityType.Field(i)
			value := entityValue.Field(i).Interface()

			// Parse the json tag
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" {
				// If no json tag, use the field name
				result[field.Name] = value
				continue
			}

			// Split the json tag
			parts := strings.Split(jsonTag, ",")
			key := parts[0]

			// If the key is "-", skip this field
			if key == "-" {
				continue
			}

			// If the key is empty, use the field name
			if key == "" {
				key = field.Name
			}

			// Check for omitempty
			if len(parts) > 1 && parts[1] == "omitempty" {
				// Only include non-zero values
				if !isZeroValue(value) {
					result[key] = value
				}
			} else {
				// Always include the field
				result[key] = value
			}
		}
	} else {
		// If it's not a struct or map, return an empty map
		log.Printf("EntityToMap: Unsupported type %v", entityType)
	}

	return result
}

// Helper function to check if a value is the zero value for its type
func isZeroValue(v interface{}) bool {
	return reflect.DeepEqual(v, reflect.Zero(reflect.TypeOf(v)).Interface())
}