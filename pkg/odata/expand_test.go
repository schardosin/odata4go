package odata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpand(t *testing.T) {
	fmt.Println("")
	fmt.Println("########## expand_test - TestExpand")
	fmt.Println("")
	r := setupTestRouter()

	t.Run("Expand all entities", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/odata/v4/Products?$expand=Category", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		values, ok := response["value"].([]interface{})
		assert.True(t, ok, "Expected 'value' to be a slice")
		assert.Len(t, values, 3, "Expected 3 entities in the response")

		for _, entity := range values {
			entityMap, ok := entity.(map[string]interface{})
			assert.True(t, ok, "Expected entity to be a map")
			category, ok := entityMap["Category"].(map[string]interface{})
			assert.True(t, ok, "Expected Category to be present and be a map")
			assert.NotEmpty(t, category["ID"], "Expected Category to have an ID")
			assert.NotEmpty(t, category["Name"], "Expected Category to have a Name")
		}
	})

	t.Run("Expand single entity", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/odata/v4/Products(1)?$expand=Category", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		category, ok := response["Category"].(map[string]interface{})
		assert.True(t, ok, "Expected Category to be present and be a map")
		assert.NotEmpty(t, category["ID"], "Expected Category to have an ID")
		assert.NotEmpty(t, category["Name"], "Expected Category to have a Name")
	})

	t.Run("Expand single entity and check field case", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/odata/v4/Products(2)?$expand=Category", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		category, ok := response["Category"].(map[string]interface{})
		assert.True(t, ok, "Expected Category to be present and be a map")
		assert.Contains(t, category, "ID", "Expected Category to have an 'ID' field")
		assert.Contains(t, category, "Name", "Expected Category to have a 'Name' field")
		assert.Equal(t, "1", category["ID"], "Expected Category ID to be '2'")
		assert.Equal(t, "Electronics", category["Name"], "Expected Category Name to be 'Books'")
	})

	t.Run("Expand all Categories with Products", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/odata/v4/Categories?$expand=Products", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		values, ok := response["value"].([]interface{})
		assert.True(t, ok, "Expected 'value' to be a slice")
		assert.NotEmpty(t, values, "Expected at least one category in the response")

		for _, entity := range values {
			categoryMap, ok := entity.(map[string]interface{})
			assert.True(t, ok, "Expected category to be a map")
			products, ok := categoryMap["Products"].([]interface{})
			assert.True(t, ok, "Expected Products to be present and be a slice")
			assert.NotEmpty(t, products, "Expected Category to have at least one Product")

			for _, product := range products {
				productMap, ok := product.(map[string]interface{})
				assert.True(t, ok, "Expected product to be a map")
				assert.NotEmpty(t, productMap["ID"], "Expected Product to have an ID")
				assert.NotEmpty(t, productMap["Name"], "Expected Product to have a Name")
			}
		}
	})

	t.Run("Expand single Category with Products", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/odata/v4/Categories(1)?$expand=Products", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		products, ok := response["Products"].([]interface{})
		assert.True(t, ok, "Expected Products to be present and be a slice")
		assert.NotEmpty(t, products, "Expected Category to have at least one Product")

		for _, product := range products {
			productMap, ok := product.(map[string]interface{})
			assert.True(t, ok, "Expected product to be a map")
			assert.NotEmpty(t, productMap["ID"], "Expected Product to have an ID")
			assert.NotEmpty(t, productMap["Name"], "Expected Product to have a Name")
		}
	})
}