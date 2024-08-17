package odata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProduct(t *testing.T) {
	fmt.Println("")
	fmt.Println("########## crud_test - TestGetProduct")
	fmt.Println("")
	r := setupTestRouter()
	
	// Test both formats: (id) and /id
	testCases := []struct {
		name string
		url  string
	}{
		{"Parentheses format", "/odata/v4/Products(1)"},
		{"Slash format", "/odata/v4/Products/1"},
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

			assert.Equal(t, "1", response["ID"], "Unexpected ID: %v", response["ID"])
			assert.Equal(t, "Product A", response["Name"], "Unexpected Name: %v", response["Name"])
		})
	}
}

func TestGetProducts(t *testing.T) {
	fmt.Println("")
	fmt.Println("########## crud_test - TestGetProducts")
	fmt.Println("")
	r := setupTestRouter()
	req, _ := http.NewRequest("GET", "/odata/v4/Products", nil)
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
}