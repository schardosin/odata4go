package odata

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type TestProducts struct {
	ID          string      `json:"ID" odata:"key"`
	Name        string      `json:"Name"`
	Description string      `json:"Description"`
	Price       float64     `json:"Price"`
	Category_ID  string      `json:"Category_ID" odata:"ref:Categories"`
	Category    *TestCategories `json:"Category,omitempty" odata:"expand:Category"`
	Supplier_ID  string      `json:"Supplier_ID" odata:"ref:Suppliers"`
    Supplier     *TestSuppliers  `json:"Supplier,omitempty" odata:"expand:Supplier"`
}

func (p TestProducts) EntityName() string {
	return "Products"
}

func (p TestProducts) GetRelationships() map[string]string {
	return map[string]string{
		"Category": "Categories",
		"Supplier": "Suppliers",
	}
}

type TestCategories struct {
	ID       string     `json:"ID" odata:"key"`
	Name     string     `json:"Name"`
	Products []TestProducts `json:"Products,omitempty" odata:"expand:Products"`
}

func (c TestCategories) EntityName() string {
	return "Categories"
}

func (c TestCategories) GetRelationships() map[string]string {
	return map[string]string{
		"Products": "Products",
	}
}

type TestSuppliers struct {
    ID       string     `json:"ID" odata:"key"`
    Name     string     `json:"Name"`
    Country  string     `json:"Country"`
    Products []TestProducts `json:"Products,omitempty" odata:"expand:Products"`
}

func (s TestSuppliers) EntityName() string {
    return "Suppliers"
}

func (s TestSuppliers) GetRelationships() map[string]string {
    return map[string]string{
        "Products": "Products",
    }
}

var testProducts = []TestProducts{
	{
		ID:          "1",
		Name:        "Product A",
		Description: "Description A",
		Price:       100.0,
		Category_ID:  "1",
		Supplier_ID: "1",
	},
	{
		ID:          "2",
		Name:        "Product B",
		Description: "Description B",
		Price:       200.0,
		Category_ID:  "1",
		Supplier_ID: "2",
	},
	{
		ID:          "3",
		Name:        "Product C",
		Description: "Description C",
		Price:       300.0,
		Category_ID:  "2",
		Supplier_ID: "1",
	},
}

var testCategories = []TestCategories{
	{ID: "1", Name: "Electronics"},
	{ID: "2", Name: "Books"},
}

var testSuppliers = []TestSuppliers{
    {ID: "1", Name: "Supplier A", Country: "USA"},
    {ID: "2", Name: "Supplier B", Country: "Canada"},
}

type TestProductHandler struct{}

func (h TestProductHandler) ExpandEntity(entity interface{}, relationshipName string, subQuery string) interface{} {
    product, ok := entity.(TestProducts)
    if !ok {
        return nil
    }

    switch relationshipName {
    case "Category":
        for _, category := range testCategories {
            if category.ID == product.Category_ID {
                return category
            }
        }
    case "Supplier":
        for _, supplier := range testSuppliers {
            if supplier.ID == product.Supplier_ID {
                return supplier
            }
        }
	}
    return nil
}

type TestCategoryExpandHandler struct{}

func (h TestCategoryExpandHandler) ExpandEntity(entity interface{}, relationshipName string, subQuery string) interface{} {
    category, ok := entity.(TestCategories)
    if !ok {
        return nil
    }

    switch relationshipName {
    case "Products":
        var categoryProducts []TestProducts
        for _, product := range testProducts {
            if product.Category_ID == category.ID {
                categoryProducts = append(categoryProducts, product)
            }
        }
        return categoryProducts
    }
    return nil
}

type TestSupplierExpandHandler struct{}

func (h TestSupplierExpandHandler) ExpandEntity(entity interface{}, relationshipName string, subQuery string) interface{} {
    supplier, ok := entity.(TestSuppliers)
    if !ok {
        return nil
    }

    switch relationshipName {
    case "Products":
        var supplierProducts []TestProducts
        for _, product := range testProducts {
            if product.Supplier_ID == supplier.ID {
                supplierProducts = append(supplierProducts, product)
            }
        }
        return supplierProducts
    }
    return nil
}

func setupTestRouter() *chi.Mux {
	r := chi.NewRouter()
	productHandler := TestProductHandler{}
	RegisterEntity(TestProducts{}, EntityHandler{
		GetEntityHandler: func(w http.ResponseWriter, r *http.Request) {
			result := ApplySkipTop(testProducts, r.URL.Query().Get("$skip"), r.URL.Query().Get("$top"))
			result = ApplyExpand(result, r.URL.Query().Get("$expand"), productHandler)
			result = ApplySelect(result, r.URL.Query().Get("$select"))
			CreateODataResponse(w, "Products", result)
		},
		GetEntityByIDHandler: func(w http.ResponseWriter, r *http.Request, id string) {
			for _, product := range testProducts {
				if product.ID == id {
					result := ApplyExpand(product, r.URL.Query().Get("$expand"), productHandler)
					result = ApplySelect(result, r.URL.Query().Get("$select"))
					CreateODataResponseSingle(w, "Products", result)
					return
				}
			}
			http.NotFound(w, r)
		},
		ExpandHandler: productHandler,
	})

	categoryHandler := TestCategoryExpandHandler{}
	RegisterEntity(TestCategories{}, EntityHandler{
		GetEntityHandler: func(w http.ResponseWriter, r *http.Request) {
			result := ApplySkipTop(testCategories, r.URL.Query().Get("$skip"), r.URL.Query().Get("$top"))
			result = ApplyExpand(result, r.URL.Query().Get("$expand"), categoryHandler)
			result = ApplySelect(result, r.URL.Query().Get("$select"))
			CreateODataResponse(w, "Categories", result)
		},
		GetEntityByIDHandler: func(w http.ResponseWriter, r *http.Request, id string) {
			for _, category := range testCategories {
				if category.ID == id {
					result := ApplyExpand(category, r.URL.Query().Get("$expand"), categoryHandler)
					result = ApplySelect(result, r.URL.Query().Get("$select"))
					CreateODataResponseSingle(w, "Categories", result)
					return
				}
			}
			http.NotFound(w, r)
		},
		ExpandHandler: categoryHandler,
	})

	supplierHandler := TestSupplierExpandHandler{}
	RegisterEntity(TestSuppliers{}, EntityHandler{
		GetEntityHandler: func(w http.ResponseWriter, r *http.Request) {
			result := ApplySkipTop(testSuppliers, r.URL.Query().Get("$skip"), r.URL.Query().Get("$top"))
			result = ApplyExpand(result, r.URL.Query().Get("$expand"), supplierHandler)
			result = ApplySelect(result, r.URL.Query().Get("$select"))
			CreateODataResponse(w, "Suppliers", result)
		},
		GetEntityByIDHandler: func(w http.ResponseWriter, r *http.Request, id string) {
			for _, supplier := range testSuppliers {
				if supplier.ID == id {
					result := ApplyExpand(supplier, r.URL.Query().Get("$expand"), supplierHandler)
					result = ApplySelect(result, r.URL.Query().Get("$select"))
					CreateODataResponseSingle(w, "Suppliers", result)
					return
				}
			}
			http.NotFound(w, r)
		},
		ExpandHandler: supplierHandler,
	})

	RegisterEntityRelationship("Products", "Category", "Categories", "one-to-one")
	RegisterEntityRelationship("Products", "Supplier", "Suppliers", "one-to-one")
	RegisterEntityRelationship("Categories", "Products", "Products", "one-to-many")
	RegisterEntityRelationship("Suppliers", "Products", "Products", "one-to-many")
	RegisterRoutes(r)
	return r
}