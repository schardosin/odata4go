package odata

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestEntity struct {
	ID          string `json:"id" odata:"key"`
	Name        string `json:"name" odata:"maxlength:30"`
	Description string `json:"description" odata:"maxlength:300"`
	Price       float64 `json:"price" odata:"precision:9,scale:2"`
	Category    *TestCategory `json:"category,omitempty" odata:"expand:Category"`
	CategoryID  string `json:"categoryId" odata:"ref:Categories"`
}

func (e TestEntity) EntityName() string {
	return "Products"
}

func (e TestEntity) GetRelationships() map[string]string {
	return map[string]string{
		"Category": "Categories",
	}
}

type TestCategory struct {
	ID       string       `json:"id" odata:"key"`
	Name     string       `json:"name" odata:"maxlength:111"`
	Products []TestEntity `json:"products,omitempty" odata:"expand:Products"`
}

func (c TestCategory) EntityName() string {
	return "Categories"
}

func (c TestCategory) GetRelationships() map[string]string {
	return map[string]string{
		"Products": "Products",
	}
}

func TestGenerateMetadata(t *testing.T) {
	// Clear existing entity types and relationships
	entityTypes = []Entity{}
	entityRelationships = make(map[string]map[string]RelationshipInfo)

	// Register test entities
	RegisterEntity(TestEntity{}, EntityHandler{})
	RegisterEntity(TestCategory{}, EntityHandler{})

	// Register relationships
	RegisterEntityRelationship("Products", "Category", "TestCategories", "one-to-one")
	RegisterEntityRelationship("Categories", "Products", "TestProducts", "one-to-many")

	metadata := GenerateMetadata()

	// Parse the generated XML
	var edmx struct {
		XMLName xml.Name `xml:"Edmx"`
		DataServices struct {
			Schema struct {
				EntityContainer struct {
					EntitySets []struct {
						Name string `xml:"Name,attr"`
						EntityType string `xml:"EntityType,attr"`
						NavigationPropertyBinding []struct {
							Path string `xml:"Path,attr"`
							Target string `xml:"Target,attr"`
						} `xml:"NavigationPropertyBinding"`
					} `xml:"EntitySet"`
				} `xml:"EntityContainer"`
				EntityTypes []struct {
					Name string `xml:"Name,attr"`
					Key struct {
						PropertyRef struct {
							Name string `xml:"Name,attr"`
						} `xml:"PropertyRef"`
					} `xml:"Key"`
					Properties []struct {
						Name string `xml:"Name,attr"`
						Type string `xml:"Type,attr"`
						Nullable string `xml:"Nullable,attr"`
						MaxLength string `xml:"MaxLength,attr"`
						Precision string `xml:"Precision,attr"`
						Scale string `xml:"Scale,attr"`
					} `xml:"Property"`
					NavigationProperties []struct {
						Name string `xml:"Name,attr"`
						Type string `xml:"Type,attr"`
						Nullable string `xml:"Nullable,attr"`
						Partner string `xml:"Partner,attr"`
						ReferentialConstraint struct {
							Property string `xml:"Property,attr"`
							ReferencedProperty string `xml:"ReferencedProperty,attr"`
						} `xml:"ReferentialConstraint"`
					} `xml:"NavigationProperty"`
				} `xml:"EntityType"`
			} `xml:"Schema"`
		} `xml:"DataServices"`
	}

	err := xml.Unmarshal([]byte(metadata), &edmx)
	assert.NoError(t, err)

	// Verify EntitySets
	assert.Len(t, edmx.DataServices.Schema.EntityContainer.EntitySets, 2)
	for _, entitySet := range edmx.DataServices.Schema.EntityContainer.EntitySets {
		assert.Contains(t, []string{"Products", "Categories"}, entitySet.Name)
		assert.Equal(t, "CatalogService."+entitySet.Name, entitySet.EntityType)
		if entitySet.Name == "Products" {
			assert.Len(t, entitySet.NavigationPropertyBinding, 1)
			assert.Equal(t, "Category", entitySet.NavigationPropertyBinding[0].Path)
			assert.Equal(t, "TestCategories", entitySet.NavigationPropertyBinding[0].Target)
		} else if entitySet.Name == "Categories" {
			assert.Len(t, entitySet.NavigationPropertyBinding, 1)
			assert.Equal(t, "Products", entitySet.NavigationPropertyBinding[0].Path)
			assert.Equal(t, "TestProducts", entitySet.NavigationPropertyBinding[0].Target)
		}
	}

	// Verify EntityTypes
	assert.Len(t, edmx.DataServices.Schema.EntityTypes, 2)
	for _, entityType := range edmx.DataServices.Schema.EntityTypes {
		assert.Contains(t, []string{"Products", "Categories"}, entityType.Name)
		assert.Equal(t, "ID", entityType.Key.PropertyRef.Name)

		if entityType.Name == "TestProducts" {
			assert.Len(t, entityType.Properties, 5)
			assert.Len(t, entityType.NavigationProperties, 1)

			// Verify specific properties
			nameProperty := findProperty(entityType.Properties, "Name")
			assert.Equal(t, "Edm.String", nameProperty.Type)
			assert.Equal(t, "30", nameProperty.MaxLength)

			priceProperty := findProperty(entityType.Properties, "Price")
			assert.Equal(t, "Edm.Decimal", priceProperty.Type)
			assert.Equal(t, "9", priceProperty.Precision)
			assert.Equal(t, "2", priceProperty.Scale)

			// Verify navigation property
			assert.Equal(t, "Category", entityType.NavigationProperties[0].Name)
			assert.Equal(t, "CatalogService.TestCategories", entityType.NavigationProperties[0].Type)
			assert.Equal(t, "Products", entityType.NavigationProperties[0].Partner)
			assert.Equal(t, "Category_ID", entityType.NavigationProperties[0].ReferentialConstraint.Property)
			assert.Equal(t, "ID", entityType.NavigationProperties[0].ReferentialConstraint.ReferencedProperty)
		} else if entityType.Name == "TestCategories" {
			assert.Len(t, entityType.Properties, 2)
			assert.Len(t, entityType.NavigationProperties, 1)

			// Verify specific properties
			nameProperty := findProperty(entityType.Properties, "Name")
			assert.Equal(t, "Edm.String", nameProperty.Type)
			assert.Equal(t, "111", nameProperty.MaxLength)

			// Verify navigation property
			assert.Equal(t, "Products", entityType.NavigationProperties[0].Name)
			assert.Equal(t, "Collection(CatalogService.TestProducts)", entityType.NavigationProperties[0].Type)
			assert.Equal(t, "Category", entityType.NavigationProperties[0].Partner)
		}
	}
}

func findProperty(properties []struct {
	Name string `xml:"Name,attr"`
	Type string `xml:"Type,attr"`
	Nullable string `xml:"Nullable,attr"`
	MaxLength string `xml:"MaxLength,attr"`
	Precision string `xml:"Precision,attr"`
	Scale string `xml:"Scale,attr"`
}, name string) struct {
	Name string `xml:"Name,attr"`
	Type string `xml:"Type,attr"`
	Nullable string `xml:"Nullable,attr"`
	MaxLength string `xml:"MaxLength,attr"`
	Precision string `xml:"Precision,attr"`
	Scale string `xml:"Scale,attr"`
} {
	for _, prop := range properties {
		if prop.Name == name {
			return prop
		}
	}
	return struct {
		Name string `xml:"Name,attr"`
		Type string `xml:"Type,attr"`
		Nullable string `xml:"Nullable,attr"`
		MaxLength string `xml:"MaxLength,attr"`
		Precision string `xml:"Precision,attr"`
		Scale string `xml:"Scale,attr"`
	}{}
}