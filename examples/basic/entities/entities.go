package entities

type Products struct {
	ID           string      `json:"ID" odata:"key"`
	Name         string      `json:"Name"`
	Description  string      `json:"Description"`
	Price        float64     `json:"Price"`
	Category_ID  string     `json:"Category_ID" odata:"ref:Categories"`
	Category     *Categories `json:"Category,omitempty" odata:"expand:Category"`
	Supplier_ID  string      `json:"Supplier_ID" oSuppliers"`
    Supplier     *Suppliers  `json:"Supplier,omitempty" odata:"expand:Supplier"`
}

func (p Products) EntityName() string {
	return "Products"
}

func (p Products) GetRelationships() map[string]string {
	return map[string]string{
		"Category": "Categories",
	}
}

type Customers struct {
	ID   string `json:"ID" odata:"key"`
	Name string `json:"Name"`
	Age  int    `json:"Age"`
}

func (c Customers) EntityName() string {
	return "Customers"
}

func (c Customers) GetRelationships() map[string]string {
	return map[string]string{}
}

type Categories struct {
	ID       string     `json:"ID" odata:"key"`
	Name     string     `json:"Name"`
	Products []Products `json:"Products,omitempty" odata:"expand:Products"`
}

func (c Categories) EntityName() string {
	return "Categories"
}

func (c Categories) GetRelationships() map[string]string {
	return map[string]string{
		"Products": "Products",
	}
}

type Suppliers struct {
    ID       string     `json:"ID" odata:"key"`
    Name     string     `json:"Name"`
    Country  string     `json:"Country"`
    Products []Products `json:"Products,omitempty" odata:"expand:Products"`
}

func (s Suppliers) EntityName() string {
    return "Suppliers"
}

func (s Suppliers) GetRelationships() map[string]string {
    return map[string]string{
        "Products": "Products",
    }
}
