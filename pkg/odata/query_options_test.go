package odata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProductsWithSkipAndTop(t *testing.T) {
	fmt.Println("")
	fmt.Println("########## query_options_test - TestGetProductsWithSkipAndTop")
	fmt.Println("")
	r := setupTestRouter()
	req, _ := http.NewRequest("GET", "/odata/v4/Products?$skip=1&$top=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected status code %d, got %d", http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "$metadata#Products", response["@odata.context"], "Unexpected @odata.context: %v", response["@odata.context"])

	values, ok := response["value"].([]interface{})
	assert.True(t, ok, "Expected value to be a slice, got %T", response["value"])
	assert.Len(t, values, 1, "Expected 1 product, got %d", len(values))

	product := values[0].(map[string]interface{})
	assert.Equal(t, "2", product["ID"], "Unexpected ID: %v", product["ID"])
	assert.Equal(t, "Product B", product["Name"], "Unexpected Name: %v", product["Name"])
}

func TestGetProductsWithSelect(t *testing.T) {
	fmt.Println("")
	fmt.Println("########## query_options_test - TestGetProductsWithSelect")
	fmt.Println("")
	r := setupTestRouter()
	req, _ := http.NewRequest("GET", "/odata/v4/Products?$select=ID,Name", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected status code %d, got %d", http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "$metadata#Products", response["@odata.context"], "Unexpected @odata.context: %v", response["@odata.context"])

	values, ok := response["value"].([]interface{})
	assert.True(t, ok, "Expected value to be a slice, got %T", response["value"])
	assert.Len(t, values, 3, "Expected 2 products, got %d", len(values))

	for _, v := range values {
		product := v.(map[string]interface{})
		assert.Len(t, product, 2, "Expected 2 fields in product, got %d: %v", len(product), product)
		assert.Contains(t, product, "ID", "Expected 'ID' field in product")
		assert.Contains(t, product, "Name", "Expected 'Name' field in product")
		assert.NotContains(t, product, "Description", "Unexpected 'Description' field in product")
	}
}

func TestGetProductWithSelect(t *testing.T) {
	fmt.Println("")
	fmt.Println("########## query_options_test - TestGetProductWithSelect")
	fmt.Println("")
	r := setupTestRouter()

	// Test both formats: (id) and /id
	testCases := []struct {
		name string
		url  string
	}{
		{"Parentheses format", "/odata/v4/Products(1)?$select=ID,Price"},
		{"Slash format", "/odata/v4/Products/1?$select=ID,Price"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tc.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "Expected status code %d, got %d", http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.Equal(t, "$metadata#Products/$entity", response["@odata.context"], "Unexpected @odata.context: %v", response["@odata.context"])

			assert.Len(t, response, 3, "Expected 3 fields in response, got %d: %v", len(response), response)
			assert.Contains(t, response, "ID", "Expected 'ID' field in response")
			assert.Contains(t, response, "Price", "Expected 'Price' field in response")
			assert.NotContains(t, response, "Name", "Unexpected 'Name' field in response")
		})
	}
}

func TestGetProductWithExpandAndSelect(t *testing.T) {
	fmt.Println("")
	fmt.Println("########## query_options_test - TestGetProductWithExpandAndSelect")
	fmt.Println("")
	r := setupTestRouter()

	req, _ := http.NewRequest("GET", "/odata/v4/Products(1)?$expand=Category,Supplier&$select=ID,Description", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected status code %d, got %d", http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "$metadata#Products/$entity", response["@odata.context"], "Unexpected @odata.context: %v", response["@odata.context"])

	assert.Contains(t, response, "ID", "Expected 'ID' field in response")
	assert.Contains(t, response, "Description", "Expected 'Description' field in response")
	assert.Contains(t, response, "Category", "Expected 'Category' field in response")
	assert.Contains(t, response, "Supplier", "Expected 'Supplier' field in response")

	category, ok := response["Category"].(map[string]interface{})
	assert.True(t, ok, "Expected Category to be a map, got %T", response["Category"])
	assert.Contains(t, category, "ID", "Expected 'ID' field in Category")
	assert.Contains(t, category, "Name", "Expected 'Name' field in Category")

	supplier, ok := response["Supplier"].(map[string]interface{})
	assert.True(t, ok, "Expected Supplier to be a map, got %T", response["Supplier"])
	assert.Contains(t, supplier, "ID", "Expected 'ID' field in Supplier")
	assert.Contains(t, supplier, "Name", "Expected 'Name' field in Supplier")
	assert.Contains(t, supplier, "Country", "Expected 'Country' field in Supplier")

	assert.NotContains(t, response, "Name", "Unexpected 'Name' field in response")
	assert.NotContains(t, response, "Price", "Unexpected 'Price' field in response")
}