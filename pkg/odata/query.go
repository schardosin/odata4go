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
		result := make([]interface{}, 0, expandedEntities.Len())

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
    var result OrderedFields
    if orderedFields, ok := entity.(OrderedFields); ok {
        result = orderedFields
    } else {
        result = EntityToOrderedFields(entity, expand)
    }

    if expand == "" {
        return result
    }

    expandParts := parseExpandQuery(expand) 
    for relationshipName, nestedExpand := range expandParts {
        if nestedExpand != "" {
            log.Printf("Processing relationship: %s with nested expand: %s", relationshipName, nestedExpand)
        }
        expandedEntity := handler.ExpandEntity(entity, relationshipName, nestedExpand)
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
                    result = append(result, struct{Key string; Value interface{}}{Key: relationshipName, Value: expandedSlice})
                } else {
                    // Handle one-to-one relationship
                    expandedOrderedFields := ApplyExpandSingle(expandedEntity, nestedExpand, nestedHandler.ExpandHandler)
                    result = append(result, struct{Key string; Value interface{}}{Key: relationshipName, Value: expandedOrderedFields})
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
			if orderedFields, ok := entities.(OrderedFields); ok {
				selectedEntity := ApplySelectSingle(orderedFields, selectedFields)
				result = append(result, selectedEntity)
			} else {
				entity := entitiesValue.Index(i).Interface()
				selectedEntity := ApplySelectSingle(EntityToOrderedFields(entity, ""), selectedFields)
				result = append(result, selectedEntity)
			}
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

	result := OrderedFields{}

	for _, field := range entity {
		if isExpandedEntity(field.Value) {
			// Always include expanded entities
			result = append(result, field)
		} else {
			for _, selectedField := range selectedFields {
				fieldParts := strings.Split(selectedField, "/")
				if strings.EqualFold(field.Key, fieldParts[0]) {
					if len(fieldParts) > 0 {
						// Handle nested selection
						switch v := field.Value.(type) {
						case OrderedFields:
							nestedResult := ApplySelectSingle(v, []string{strings.Join(fieldParts[1:], "/")})
							result = append(result, struct{Key string; Value interface{}}{Key: field.Key, Value: nestedResult})
						case []OrderedFields:
							nestedSlice := make([]OrderedFields, len(v))
							for i, item := range v {
								nestedSlice[i] = ApplySelectSingle(item, []string{strings.Join(fieldParts[1:], "/")})
							}
							result = append(result, struct{Key string; Value interface{}}{Key: field.Key, Value: nestedSlice})
						default:
							// If it's not an OrderedFields or []OrderedFields, just add it as is
							result = append(result, field)
						}
					} else {
						result = append(result, field)
					}
					break
				}
			}
		}
	}

	return result
}

func isExpandedEntity(value interface{}) bool {
	switch v := value.(type) {
	case OrderedFields, []OrderedFields:
		return true
	case map[string]interface{}, []map[string]interface{}:
		return true
	case []interface{}:
		// Check if it's a slice of OrderedFields or map[string]interface{}
		if len(v) > 0 {
			switch v[0].(type) {
			case OrderedFields, map[string]interface{}:
				return true
			}
		}
	}
	return false
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