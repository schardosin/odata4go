package odata

import (
	"net/http"
	"reflect"
	"strings"
)

func handleGetMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("OData-Version", "4.0")
	metadata := generateMetadata()
	w.Write([]byte(metadata))
}

func generateMetadata() string {
	edm := `<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
        <edmx:Reference Uri="https://sap.github.io/odata-vocabularies/vocabularies/Common.xml">
            <edmx:Include Alias="Common" Namespace="com.sap.vocabularies.Common.v1"/>
        </edmx:Reference>
        <edmx:Reference Uri="https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Core.V1.xml">
            <edmx:Include Alias="Core" Namespace="Org.OData.Core.V1"/>
        </edmx:Reference>
        <edmx:DataServices>
            <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="CatalogService">
                <EntityContainer Name="EntityContainer">`

	for _, entityType := range entityTypes {
		entitySetName := entityType.EntityName()
		edm += `<EntitySet Name="` + entitySetName + `" EntityType="CatalogService.` + entitySetName + `">`
		relationships := entityRelationships[entitySetName]
		for relationshipName, relInfo := range relationships {
			edm += `<NavigationPropertyBinding Path="` + relationshipName + `" Target="` + relInfo.TargetEntity + `"/>`
		}
		edm += `</EntitySet>`
	}

	edm += `</EntityContainer>`

	for _, entityType := range entityTypes {
		edm += generateEntityTypeMetadata(entityType)
	}

	edm += `</Schema>
        </edmx:DataServices>
    </edmx:Edmx>`
	return edm
}

func generateEntityTypeMetadata(entityType Entity) string {
	entityTypeValue := reflect.TypeOf(entityType)
	entityTypeName := entityType.EntityName()
	entityMetadata := `<EntityType Name="` + entityTypeName + `">`

	// Add Key
	entityMetadata += `<Key>`
	for i := 0; i < entityTypeValue.NumField(); i++ {
		field := entityTypeValue.Field(i)
		if hasODataTag(field, "key") {
			entityMetadata += `<PropertyRef Name="` + field.Name + `"/>`
		}
	}
	entityMetadata += `</Key>`

	// Add Properties and Navigation Properties
	for i := 0; i < entityTypeValue.NumField(); i++ {
		field := entityTypeValue.Field(i)
		if isNavigationProperty(field) {
			entityMetadata += generateNavigationPropertyMetadata(field, entityTypeName, entityTypeValue)
		} else {
			entityMetadata += generatePropertyMetadata(field)
		}
	}

	entityMetadata += `</EntityType>`
	return entityMetadata
}

func generatePropertyMetadata(field reflect.StructField) string {
	edmType := mapGoTypeToEdmType(field.Type)
	
	// Set nullable to true by default
	nullable := "true"
	
	// Check if the field is a key or explicitly set as not null
	if hasODataTag(field, "key") || hasODataTag(field, "notnull") {
		nullable = "false"
	}

	metadata := `<Property Name="` + field.Name + `" Type="Edm.` + edmType + `"`
	
	// Only add Nullable attribute if it's false
	if nullable == "false" {
		metadata += ` Nullable="false"`
	}

	// Handle OData tags
	oDataTags := getODataTags(field)
	for key, value := range oDataTags {
		switch key {
		case "maxlength":
			metadata += ` MaxLength="` + value + `"`
		case "precision":
			metadata += ` Precision="` + value + `"`
		case "scale":
			metadata += ` Scale="` + value + `"`
		}
	}

	metadata += "/>"
	return metadata
}

func generateNavigationPropertyMetadata(field reflect.StructField, parentTypeName string, parentType reflect.Type) string {
	relationships := entityRelationships[parentTypeName]
	relInfo, exists := relationships[field.Name]
	if !exists {
		return ""
	}

	metadata := `<NavigationProperty Name="` + field.Name + `" Type="`
	if relInfo.Type == "one-to-many" {
		metadata += `Collection(CatalogService.` + relInfo.TargetEntity + `)`
	} else {
		metadata += `CatalogService.` + relInfo.TargetEntity
	}
	metadata += `"`

	// Add Partner if it exists
	for _, partnerRelationships := range entityRelationships {
		for partnerRelationship, partnerRelInfo := range partnerRelationships {
			if partnerRelInfo.TargetEntity == parentTypeName {
				metadata += ` Partner="` + partnerRelationship + `"`
				break
			}
		}
	}

	metadata += ">"

	// Add ReferentialConstraint
	refConstraintField := field.Name + "_ID"
	if _, ok := parentType.FieldByName(refConstraintField); ok {
		metadata += `<ReferentialConstraint Property="` + refConstraintField + `" ReferencedProperty="ID"/>`
	}

	metadata += `</NavigationProperty>`
	return metadata
}

func mapGoTypeToEdmType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "String"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "Int32"
	case reflect.Float32, reflect.Float64:
		return "Decimal"
	case reflect.Bool:
		return "Boolean"
	default:
		return "String"
	}
}

func isNullable(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Map
}

func isNavigationProperty(field reflect.StructField) bool {
	return field.Type.Kind() == reflect.Ptr || field.Type.Kind() == reflect.Slice
}

func hasODataTag(field reflect.StructField, tag string) bool {
	oDataTag := field.Tag.Get("odata")
	return strings.Contains(oDataTag, tag)
}

func getODataTags(field reflect.StructField) map[string]string {
	tags := make(map[string]string)
	oDataTag := field.Tag.Get("odata")
	if oDataTag == "" {
		return tags
	}

	for _, tag := range strings.Split(oDataTag, ",") {
		parts := strings.SplitN(tag, ":", 2)
		if len(parts) == 2 {
			tags[strings.ToLower(parts[0])] = parts[1]
		}
	}
	return tags
}
