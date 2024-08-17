# odata4go

odata4go is a Go library for implementing OData v4 APIs. It provides a simple way to create OData-compliant endpoints in your Go applications.

## Installation

To install odata4go, use `go get`:

```bash
go get github.com/schardosin/odata4go
```

## Usage

Here's a basic example of how to use odata4go by using the example implementation in the project:

```go
package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/schardosin/odata4go/examples/basic/routes"
	"github.com/schardosin/odata4go/pkg/odata"
)

func main() {
	r := chi.NewRouter()
	routes.SetupRoutes()
	odata.RegisterRoutes(r)

	log.Println("Server is running on http://localhost:8000")
	err := http.ListenAndServe(":8000", r)
	if err != nil {
		log.Fatal("ListenAndServe error: ", err)
	}
}
```

For more detailed examples, check the `examples` directory in the repository.

## Features

- Support for OData v4 query options ($select, $expand, $filter, etc.)
- Dynamic metadata generation
- Entity relationships
- Customizable entity handlers

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.