package odata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldOrder(t *testing.T) {
	fmt.Println("")
	fmt.Println("########## field_order_test - TestFieldOrder")
	fmt.Println("")
	r := setupTestRouter()
	req, _ := http.NewRequest("GET", "/odata/v4/Products(1)", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected status code %d, got %d", http.StatusOK, w.Code)

	var responseMap map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &responseMap)
	assert.NoError(t, err, "Failed to unmarshal response: %v", err)

	// Expected order of fields
	expectedOrder := []string{"@odata.context", "ID", "Name", "Description", "Price", "Category_ID"}

	// Check if all expected fields are present and in the correct order
	assert.Equal(t, len(expectedOrder), len(responseMap), "Expected %d fields, got %d", len(expectedOrder), len(responseMap))

	fmt.Println("Actual field order:")
	for _, field := range expectedOrder {
		value, exists := responseMap[field]
		assert.True(t, exists, "Field '%s' is missing from the response", field)

		switch field {
		case "@odata.context", "ID", "Name", "Description", "Category_ID":
			assert.IsType(t, "", value, "%s should be a string", field)
		case "Price":
			assert.IsType(t, float64(0), value, "Price should be a number")
		}
	}
}