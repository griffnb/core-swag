package main

import (
	swag "github.com/griffnb/core-swag/internal/legacy_files"
	"github.com/griffnb/core-swag/testing/testdata/delims/api"
	_ "github.com/griffnb/core-swag/testing/testdata/delims/docs"
)

func ReadDoc() string {
	doc, _ := swag.ReadDoc("CustomDelims")
	return doc
}

// @title Swagger Example API
// @version 1.0
// @description Testing custom template delimeters
// @termsOfService http://swagger.io/terms/

func main() {
	api.MyFunc()
}
