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
	result := EntityToOrderedFields(entity, expand)
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
			for i, field := range result {
				if field.Key == relationshipName {
					result = append(result[:i], result[i+1:]...)
					break
				}
			}

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
					// Convert each item in the slice to OrderedFields
					expandedSlice := make([]OrderedFields, expandedValue.Len())
					for i := 0; i < expandedValue.Len(); i++ {
						expandedSlice[i] = ApplyExpandSingle(expandedValue.Index(i).Interface(), nestedExpand, nestedHandler.ExpandHandler)
					}
					// Add the expanded result
					result = append(result, struct{Key string; Value interface{}}{relationshipName, expandedSlice})
				} else {
					// Handle one-to-one relationship
					expandedOrderedFields := ApplyExpandSingle(expandedEntity, nestedExpand, nestedHandler.ExpandHandler)
					result = append(result, struct{Key string; Value interface{}}{relationshipName, expandedOrderedFields})
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
		result := make([]OrderedFields, 0, entitiesValue.Len())
		for i := 0; i < entitiesValue.Len(); i++ {
			entity := entitiesValue.Index(i).Interface()
			selectedEntity := ApplySelectSingle(EntityToOrderedFields(entity, ""), selectedFields)
			result = append(result, selectedEntity)
		}
		return result
	} else {
		log.Printf("ApplySelect: Processing single entity of type: %s", entitiesType.Name())
		result := ApplySelectSingle(EntityToOrderedFields(entities, ""), selectedFields)
		return result
	}
}

func ApplySelectSingle(entity OrderedFields, selectedFields []string) OrderedFields {
	if len(selectedFields) == 0 {
		return entity
	}

	result := make(OrderedFields, 0, len(selectedFields))

	for _, kv := range entity {
		if isExpandedEntity(kv.Value) {
			// Always include expanded entities
			result = append(result, kv)
		} else {
			for _, field := range selectedFields {
				fieldParts := strings.Split(field, "/")
				if strings.EqualFold(kv.Key, fieldParts[0]) {
					if len(fieldParts) > 1 {
						// Handle nested selection
						switch v := kv.Value.(type) {
						case OrderedFields:
							nestedResult := ApplySelectSingle(v, []string{strings.Join(fieldParts[1:], "/")})
							result = append(result, struct{Key string; Value interface{}}{kv.Key, nestedResult})
						case []OrderedFields:
							nestedSlice := make([]OrderedFields, len(v))
							for i, item := range v {
								nestedSlice[i] = ApplySelectSingle(item, []string{strings.Join(fieldParts[1:], "/")})
							}
							result = append(result, struct{Key string; Value interface{}}{kv.Key, nestedSlice})
						default:
							// If it's not an OrderedFields or []OrderedFields, just add it as is
							result = append(result, kv)
						}
					} else {
						result = append(result, kv)
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
	case OrderedFields, []OrderedFields:
		return true
	default:
		return false
	}
}