package main

import (
	swag "github.com/griffnb/core-swag/internal/legacy_files"
	"github.com/griffnb/core-swag/testing/testdata/quotes/api"
	_ "github.com/griffnb/core-swag/testing/testdata/quotes/docs"
)

func ReadDoc() string {
	doc, _ := swag.ReadDoc()
	return doc
}

// @title Swagger Example API
// @version 1.0
// @description.markdown
// @tag.name api
// @tag.description.markdown
// @termsOfService http://swagger.io/terms/

func main() {
	api.RandomFunc()
}
