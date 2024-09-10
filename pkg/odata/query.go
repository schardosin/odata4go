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
	if expand == "" || handler == nil {
		return entities
	}

	log.Printf("ApplyExpand called with expand: %s", expand)

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

    // Convert entity to OrderedFields if it's not already
    switch v := entity.(type) {
    case OrderedFields:
        result = v
    default:
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
        expandedEntity := handler.ExpandEntity(result, relationshipName, nestedExpand)
        if expandedEntity != nil {
            // Remove the existing field if it exists
            for i, field := range result.Fields {
                if field.Key == relationshipName {
                    result.Fields = append(result.Fields[:i], result.Fields[i+1:]...)
                    break
                }
            }

            // Convert the expanded entity to OrderedFields if necessary
            var expandedOrderedFields interface{}
            if reflect.TypeOf(expandedEntity).Kind() == reflect.Slice {
                log.Printf("Expanded entity is a slice")
                expandedSlice := reflect.ValueOf(expandedEntity)
                expandedOrderedFieldsSlice := make([]OrderedFields, expandedSlice.Len())
                for i := 0; i < expandedSlice.Len(); i++ {
                    nestedHandler := getHandlerForEntity(expandedSlice.Index(i).Interface())
                    expandedOrderedFieldsSlice[i] = ApplyExpandSingle(expandedSlice.Index(i).Interface(), nestedExpand, nestedHandler)
                }
                expandedOrderedFields = expandedOrderedFieldsSlice
            } else {
                log.Printf("Expanded entity is not a slice")
                nestedHandler := getHandlerForEntity(expandedEntity)
                expandedOrderedFields = ApplyExpandSingle(expandedEntity, nestedExpand, nestedHandler)
            }

            // Add the expanded result
            result.Fields = append(result.Fields, struct{Key string; Value interface{}}{Key: relationshipName, Value: expandedOrderedFields})
        } else {
            log.Printf("ExpandEntity returned nil for %s", relationshipName)
        }
    }

    return result
}

func getHandlerForEntity(entity interface{}) ExpandHandler {
    if orderedFields, ok := entity.(OrderedFields); ok {
        if handler, ok := GetEntityHandler(orderedFields.EntityName); ok {
            return handler.ExpandHandler
        }
    } else if entity, ok := entity.(Entity); ok {
        entityName := entity.EntityName()
        if handler, ok := GetEntityHandler(entityName); ok {
            return handler.ExpandHandler
        }
    }
    log.Printf("No specific handler found for entity type %T, using default handler", entity)
    return DefaultExpandHandler{}
}

func parseExpandQuery(url string) map[string]string {
    expandParts := make(map[string]string)
    
    // Extract the $expand part
    expandStart := strings.Index(url, "$expand=")
    if expandStart == -1 {
        return expandParts // No $expand found
    }
    
    expandStart += 8 // Move past "$expand="
    expandEnd := len(url)
    nestedLevel := 0
    
    for i := expandStart; i < len(url); i++ {
        if url[i] == '(' {
            nestedLevel++
        } else if url[i] == ')' {
            nestedLevel--
        } else if url[i] == '&' && nestedLevel == 0 {
            expandEnd = i
            break
        }
    }
    
    expand := url[expandStart:expandEnd]
    
    // Now parse the expand part
    currentKey := ""
    nestedLevel = 0
    var currentValue strings.Builder

    for _, char := range expand {
        switch char {
        case '(':
            if nestedLevel == 0 {
                // Don't add the opening parenthesis at the top level
                nestedLevel++
            } else {
                currentValue.WriteRune(char)
                nestedLevel++
            }
        case ')':
            nestedLevel--
            if nestedLevel > 0 {
                currentValue.WriteRune(char)
            }
            if nestedLevel == 0 {
                expandParts[currentKey] = strings.TrimSpace(currentValue.String())
                currentKey = ""
                currentValue.Reset()
            }
        case ',':
            if nestedLevel == 0 {
                if currentKey != "" {
                    expandParts[currentKey] = strings.TrimSpace(currentValue.String())
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
        expandParts[currentKey] = strings.TrimSpace(currentValue.String())
    }

    return expandParts
}

func parseNestedExpand(nestedExpand string) string {
	if strings.HasPrefix(nestedExpand, "$expand=") {
		nestedExpand = strings.TrimPrefix(nestedExpand, "$expand=")
	}
	return nestedExpand
}

func ParseSelect(selectQuery string) string {
    var result strings.Builder
    inParentheses := 0
    selectFound := false
    
    for i := 0; i < len(selectQuery); i++ {
        char := selectQuery[i]
        
        switch char {
        case '(':
            inParentheses++
        case ')':
            inParentheses--
        case '$':
            if inParentheses == 0 && i+7 < len(selectQuery) && selectQuery[i:i+7] == "$select" {
                selectFound = true
                i += 7 // Skip "$select="
                for i < len(selectQuery) && selectQuery[i] != '=' {
                    i++
                }
                continue
            }
        case '&':
            if inParentheses == 0 && selectFound {
                return strings.TrimSpace(result.String())
            }
        }
        
        if selectFound && inParentheses == 0 {
            result.WriteByte(char)
        }
    }
    
    return strings.TrimSpace(result.String())
}

func ApplySelect(entities interface{}, selectQuery string) interface{} {
	if selectQuery == "" {
		return entities
	}

	parsedSelectQuery := ParseSelect(selectQuery)
	if parsedSelectQuery == "" {
		return entities
	}

	selectedFields := strings.Split(parsedSelectQuery, ",")

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
				selectedEntity := ApplySelectSingle(entity.(OrderedFields), selectedFields)
				result = append(result, selectedEntity)
			}
		}
		return result
	} else if entitiesType == reflect.TypeOf(OrderedFields{}) {
		result := ApplySelectSingle(entities.(OrderedFields), selectedFields)
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

	result := OrderedFields{EntityName: entity.EntityName}

	for _, field := range entity.Fields {
		log.Printf("ApplySelectSingle: Processing field: %s, Type: %T, Value: %v", field.Key, field.Value, field.Value)
		if isExpandedEntity(field.Value) {
			log.Printf("ApplySelectSingle: Field %s is an expanded entity", field.Key)
			result.Fields = append(result.Fields, field)
		} else {
			for _, selectedField := range selectedFields {
				fieldParts := strings.Split(selectedField, "/")
				if strings.EqualFold(field.Key, fieldParts[0]) {
					if len(fieldParts) > 1 {
						log.Printf("ApplySelectSingle: Handling nested selection for field: %s", field.Key)
						switch v := field.Value.(type) {
						case OrderedFields:
							log.Printf("ApplySelectSingle: Field %s is OrderedFields", field.Key)
							nestedResult := ApplySelectSingle(v, []string{strings.Join(fieldParts[1:], "/")})
							result.Fields = append(result.Fields, struct{Key string; Value interface{}}{Key: field.Key, Value: nestedResult})
						case []OrderedFields:
							log.Printf("ApplySelectSingle: Field %s is []OrderedFields", field.Key)
							nestedSlice := make([]OrderedFields, len(v))
							for i, item := range v {
								nestedSlice[i] = ApplySelectSingle(item, []string{strings.Join(fieldParts[1:], "/")})
							}
							result.Fields = append(result.Fields, struct{Key string; Value interface{}}{Key: field.Key, Value: nestedSlice})
						default:
							log.Printf("ApplySelectSingle: Field %s is not OrderedFields or []OrderedFields, Type: %T", field.Key, v)
							result.Fields = append(result.Fields, field)
						}
					} else {
						log.Printf("ApplySelectSingle: Adding field %s as is, Type: %T", field.Key, field.Value)
						result.Fields = append(result.Fields, field)
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