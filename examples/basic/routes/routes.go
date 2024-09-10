package routes

import (
	"net/http"

	"github.com/schardosin/odata4go/examples/basic/entities"
	"github.com/schardosin/odata4go/pkg/odata"
)

var products = []entities.Products{
    {
        ID:          "1",
        Name:        "Product A",
        Description: "Description A",
        Price:       100.0,
        Category_ID: "1",
        Supplier_ID: "1",
    },
    {
        ID:          "2",
        Name:        "Product B",
        Description: "Description B",
        Price:       200.0,
        Category_ID: "1",
        Supplier_ID: "2",
    },
    {
        ID:          "3",
        Name:        "Product C",
        Description: "Description C",
        Price:       300.0,
        Category_ID: "2",
        Supplier_ID: "1",
    },
}

var categories = []entities.Categories{
    {ID: "1", Name: "Electronics"},
    {ID: "2", Name: "Books"},
}

var customers = []entities.Customers{
    {ID: "1", Name: "John Doe", Age: 30},
    {ID: "2", Name: "Jane Smith", Age: 25},
}

var suppliers = []entities.Suppliers{
    {ID: "1", Name: "Supplier A", Country: "USA"},
    {ID: "2", Name: "Supplier B", Country: "Canada"},
}

type ProductExpandHandler struct{}

func (h ProductExpandHandler) ExpandEntity(entity odata.OrderedFields, relationshipName string, subQuery string) interface{} {
    var product entities.Products
    for _, field := range entity.Fields {
        switch field.Key {
        case "ID":
            product.ID = field.Value.(string)
        case "Category_ID":
            product.Category_ID = field.Value.(string)
        case "Supplier_ID":
            product.Supplier_ID = field.Value.(string)
        }
    }

    var result interface{}
    switch relationshipName {
    case "Category":
        for _, category := range categories {
            if category.ID == product.Category_ID {
                result = odata.ApplySelect(category, subQuery)
                return result
            }
        }
    case "Supplier":
        for _, supplier := range suppliers {
            if supplier.ID == product.Supplier_ID {
                result = odata.ApplySelect(supplier, subQuery)
                return result

                break
            }
        }
    }

    return nil
}

type CategoryExpandHandler struct{}

func (h CategoryExpandHandler) ExpandEntity(entity odata.OrderedFields, relationshipName string, subQuery string) interface{} {
    var categoryID string
    for _, field := range entity.Fields {
        if field.Key == "ID" {
            categoryID = field.Value.(string)
            break
        }
    }

    var result interface{}
    switch relationshipName {
    case "Products":
        var categoryProducts []entities.Products
        for _, product := range products {
            if product.Category_ID == categoryID {
                result = odata.ApplySelect(product, subQuery)
                return result
            }
        }
        result = categoryProducts
    }

    return nil
}

type SupplierExpandHandler struct{}

func (h SupplierExpandHandler) ExpandEntity(entity odata.OrderedFields, relationshipName string, subQuery string) interface{} {
    var supplierID string
    for _, field := range entity.Fields {
        if field.Key == "ID" {
            supplierID = field.Value.(string)
            break
        }
    }

    var result interface{}
    switch relationshipName {
    case "Products":
        var supplierProducts []entities.Products
        for _, product := range products {
            if product.Supplier_ID == supplierID {
                result = odata.ApplySelect(product, subQuery)
                return result
            }
        }
        result = supplierProducts
    }

    return nil
}

func SetupRoutes() {
    productHandler := ProductExpandHandler{}
    odata.RegisterEntity(entities.Products{}, odata.EntityHandler{
        GetEntityHandler: func(w http.ResponseWriter, r *http.Request) {
            result := odata.ApplySkipTop(products, r.URL.Query().Get("$skip"), r.URL.Query().Get("$top"))
            result = odata.ApplyExpand(result, r.URL.RawQuery, productHandler)
            result = odata.ApplySelect(result, r.URL.RawQuery)
            odata.CreateODataResponse(w, "Products", result)
        },
        GetEntityByIDHandler: func(w http.ResponseWriter, r *http.Request, id string) {
            for _, product := range products {
                if product.ID == id {
                    result := odata.ApplyExpand(product, r.URL.RawQuery, productHandler)
                    result = odata.ApplySelect(result, r.URL.RawQuery)
                    odata.CreateODataResponseSingle(w, "Products", result)
                    return
                }
            }
            http.NotFound(w, r)
        },
        ExpandHandler: productHandler,
    })

    categoryHandler := CategoryExpandHandler{}
    odata.RegisterEntity(entities.Categories{}, odata.EntityHandler{
        GetEntityHandler: func(w http.ResponseWriter, r *http.Request) {
            result := odata.ApplySkipTop(categories, r.URL.Query().Get("$skip"), r.URL.Query().Get("$top"))
            result = odata.ApplyExpand(result, r.URL.RawQuery, categoryHandler)
            result = odata.ApplySelect(result, r.URL.RawQuery)
            odata.CreateODataResponse(w, "Categories", result)
        },
        GetEntityByIDHandler: func(w http.ResponseWriter, r *http.Request, id string) {
            for _, category := range categories {
                if category.ID == id {
                    result := odata.ApplyExpand(category, r.URL.RawQuery, categoryHandler)
                    result = odata.ApplySelect(result, r.URL.RawQuery)
                    odata.CreateODataResponseSingle(w, "Categories", result)
                    return
                }
            }
            http.NotFound(w, r)
        },
        ExpandHandler: categoryHandler,
    })

    supplierHandler := SupplierExpandHandler{}
    odata.RegisterEntity(entities.Suppliers{}, odata.EntityHandler{
        GetEntityHandler: func(w http.ResponseWriter, r *http.Request) {
            result := odata.ApplySkipTop(suppliers, r.URL.Query().Get("$skip"), r.URL.Query().Get("$top"))
            result = odata.ApplyExpand(result, r.URL.RawQuery, supplierHandler)
            result = odata.ApplySelect(result, r.URL.RawQuery)
            odata.CreateODataResponse(w, "Suppliers", result)
        },
        GetEntityByIDHandler: func(w http.ResponseWriter, r *http.Request, id string) {
            for _, supplier := range suppliers {
                if supplier.ID == id {
                    result := odata.ApplyExpand(supplier, r.URL.RawQuery, supplierHandler)
                    result = odata.ApplySelect(result, r.URL.RawQuery)
                    odata.CreateODataResponseSingle(w, "Suppliers", result)
                    return
                }
            }
            http.NotFound(w, r)
        },
        ExpandHandler: supplierHandler,
    })

    odata.RegisterEntity(entities.Customers{}, odata.EntityHandler{
        GetEntityHandler: func(w http.ResponseWriter, r *http.Request) {
            result := odata.ApplySkipTop(customers, r.URL.Query().Get("$skip"), r.URL.Query().Get("$top"))
            result = odata.ApplySelect(result, r.URL.RawQuery)
            odata.CreateODataResponse(w, "Customers", result)
        },
        GetEntityByIDHandler: func(w http.ResponseWriter, r *http.Request, id string) {
            for _, customer := range customers {
                if customer.ID == id {
                    result := odata.ApplySelect(customer, r.URL.RawQuery)
                    odata.CreateODataResponseSingle(w, "Customers", result)
                    return
                }
            }
            http.NotFound(w, r)
        },
    })

    odata.RegisterEntityRelationship("Products", "Category", "Categories", "one-to-one")
    odata.RegisterEntityRelationship("Products", "Supplier", "Suppliers", "one-to-one")
    odata.RegisterEntityRelationship("Categories", "Products", "Products", "one-to-many")
    odata.RegisterEntityRelationship("Suppliers", "Products", "Products", "one-to-many")
}