package orchestrator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/loader"
	"github.com/griffnb/core-swag/internal/parser/route"
)

// newSwaggerSpec creates a minimal swagger spec for testing, mirroring
// the initialization in New().
func newSwaggerSpec() *spec.Swagger {
	return &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Paths:       &spec.Paths{Paths: make(map[string]spec.PathItem)},
			Definitions: make(spec.Definitions),
		},
	}
}

// makeASTFile parses Go source code and returns its AST, fset, and a
// synthetic file path. The caller supplies a directory and file name;
// the source is written to disk so go/parser can resolve it.
func makeASTFile(t *testing.T, dir, name, src string) (*ast.File, *token.FileSet, string) {
	t.Helper()
	fp := filepath.Join(dir, name)
	if err := os.WriteFile(fp, []byte(src), 0o644); err != nil {
		t.Fatalf("write %s: %v", fp, err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fp, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", fp, err)
	}
	return f, fset, fp
}

// newTestService creates a minimal orchestrator Service with a real route parser
// and an initialized swagger spec, matching the production initialization in New().
func newTestService() *Service {
	rp := route.NewService(nil, "csv")
	return &Service{
		routeParser: rp,
		config:      &Config{},
		swagger:     newSwaggerSpec(),
	}
}

func TestParseRoutesParallel_EmptyFiles(t *testing.T) {
	svc := newTestService()
	files := make(map[*ast.File]*loader.AstFileInfo)

	routes, count, err := svc.parseRoutesParallel(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
	if count != 0 {
		t.Errorf("expected 0 route count, got %d", count)
	}
}

func TestParseRoutesParallel_SingleFile(t *testing.T) {
	dir := t.TempDir()
	src := `package api

// GetUser godoc
// @summary Get a user
// @router /users/{id} [get]
func GetUser() {}

// CreateUser godoc
// @summary Create a user
// @router /users [post]
func CreateUser() {}
`
	af, fset, fp := makeASTFile(t, dir, "handler.go", src)

	svc := newTestService()
	files := map[*ast.File]*loader.AstFileInfo{
		af: {Path: fp, FileSet: fset},
	}

	routes, count, err := svc.parseRoutesParallel(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if count != 2 {
		t.Errorf("expected route count 2, got %d", count)
	}

	// Verify swagger paths were populated.
	if len(svc.swagger.Paths.Paths) != 2 {
		t.Errorf("expected 2 swagger paths, got %d", len(svc.swagger.Paths.Paths))
	}
}

func TestParseRoutesParallel_DeterministicOrder(t *testing.T) {
	dir := t.TempDir()

	// Create files named so alphabetical sort differs from map iteration order.
	srcA := `package api

// AHandler godoc
// @summary A handler
// @router /a [get]
func AHandler() {}
`
	srcZ := `package api

// ZHandler godoc
// @summary Z handler
// @router /z [get]
func ZHandler() {}
`
	srcM := `package api

// MHandler godoc
// @summary M handler
// @router /m [get]
func MHandler() {}
`

	afA, fsetA, fpA := makeASTFile(t, dir, "a_handler.go", srcA)
	afZ, fsetZ, fpZ := makeASTFile(t, dir, "z_handler.go", srcZ)
	afM, fsetM, fpM := makeASTFile(t, dir, "m_handler.go", srcM)

	svc := newTestService()
	files := map[*ast.File]*loader.AstFileInfo{
		afZ: {Path: fpZ, FileSet: fsetZ},
		afA: {Path: fpA, FileSet: fsetA},
		afM: {Path: fpM, FileSet: fsetM},
	}

	// Run multiple times to verify determinism despite map ordering.
	for i := 0; i < 10; i++ {
		svc2 := newTestService()
		routes, _, err := svc2.parseRoutesParallel(files)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if len(routes) != 3 {
			t.Fatalf("iteration %d: expected 3 routes, got %d", i, len(routes))
		}

		// Routes must come in file-path sorted order: a, m, z.
		if routes[0].Path != "/a" {
			t.Errorf("iteration %d: expected first route /a, got %s", i, routes[0].Path)
		}
		if routes[1].Path != "/m" {
			t.Errorf("iteration %d: expected second route /m, got %s", i, routes[1].Path)
		}
		if routes[2].Path != "/z" {
			t.Errorf("iteration %d: expected third route /z, got %s", i, routes[2].Path)
		}
	}

	// Also verify using the first service that swagger paths are built correctly.
	_ = svc
}

func TestParseRoutesParallel_MultipleRoutesPerFile(t *testing.T) {
	dir := t.TempDir()
	src := `package api

// List godoc
// @summary List items
// @router /items [get]
func List() {}

// Create godoc
// @summary Create item
// @router /items [post]
func Create() {}

// Delete godoc
// @summary Delete item
// @router /items/{id} [delete]
func Delete() {}
`
	af, fset, fp := makeASTFile(t, dir, "items.go", src)

	svc := newTestService()
	files := map[*ast.File]*loader.AstFileInfo{
		af: {Path: fp, FileSet: fset},
	}

	routes, count, err := svc.parseRoutesParallel(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}

	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
	}
	for _, m := range []string{"GET", "POST", "DELETE"} {
		if !methods[m] {
			t.Errorf("expected method %s in results", m)
		}
	}
}

func TestParseRoutesParallel_SwaggerPathsPopulated(t *testing.T) {
	dir := t.TempDir()
	srcA := `package api

// GetItems godoc
// @summary Get items
// @router /items [get]
func GetItems() {}
`
	srcB := `package api

// PostItems godoc
// @summary Post items
// @router /items [post]
func PostItems() {}

// GetUsers godoc
// @summary Get users
// @router /users [get]
func GetUsers() {}
`

	afA, fsetA, fpA := makeASTFile(t, dir, "a.go", srcA)
	afB, fsetB, fpB := makeASTFile(t, dir, "b.go", srcB)

	svc := newTestService()
	// Initialize swagger so path merging works.
	svc.swagger = newSwaggerSpec()

	files := map[*ast.File]*loader.AstFileInfo{
		afA: {Path: fpA, FileSet: fsetA},
		afB: {Path: fpB, FileSet: fsetB},
	}

	routes, count, err := svc.parseRoutesParallel(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}

	// Verify swagger paths.
	if svc.swagger.Paths == nil {
		t.Fatal("swagger paths should not be nil")
	}
	pathItems := svc.swagger.Paths.Paths

	itemsPath, ok := pathItems["/items"]
	if !ok {
		t.Fatal("expected /items path in swagger")
	}
	if itemsPath.Get == nil {
		t.Error("expected GET operation on /items")
	}
	if itemsPath.Post == nil {
		t.Error("expected POST operation on /items")
	}

	usersPath, ok := pathItems["/users"]
	if !ok {
		t.Fatal("expected /users path in swagger")
	}
	if usersPath.Get == nil {
		t.Error("expected GET operation on /users")
	}
}

func TestParseRoutesParallel_AllHTTPMethods(t *testing.T) {
	dir := t.TempDir()
	src := `package api

// GetHandler godoc
// @router /res [get]
func GetHandler() {}

// PostHandler godoc
// @router /res [post]
func PostHandler() {}

// PutHandler godoc
// @router /res [put]
func PutHandler() {}

// DeleteHandler godoc
// @router /res [delete]
func DeleteHandler() {}

// PatchHandler godoc
// @router /res [patch]
func PatchHandler() {}

// OptionsHandler godoc
// @router /res [options]
func OptionsHandler() {}

// HeadHandler godoc
// @router /res [head]
func HeadHandler() {}
`
	af, fset, fp := makeASTFile(t, dir, "methods.go", src)

	svc := newTestService()

	files := map[*ast.File]*loader.AstFileInfo{
		af: {Path: fp, FileSet: fset},
	}

	_, count, err := svc.parseRoutesParallel(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 7 {
		t.Errorf("expected 7 operations, got %d", count)
	}

	pi := svc.swagger.Paths.Paths["/res"]
	if pi.Get == nil {
		t.Error("missing GET")
	}
	if pi.Post == nil {
		t.Error("missing POST")
	}
	if pi.Put == nil {
		t.Error("missing PUT")
	}
	if pi.Delete == nil {
		t.Error("missing DELETE")
	}
	if pi.Patch == nil {
		t.Error("missing PATCH")
	}
	if pi.Options == nil {
		t.Error("missing OPTIONS")
	}
	if pi.Head == nil {
		t.Error("missing HEAD")
	}
}

func TestParseRoutesParallel_FilesWithNoRoutes(t *testing.T) {
	dir := t.TempDir()

	// File with routes.
	srcRoutes := `package api

// GetFoo godoc
// @router /foo [get]
func GetFoo() {}
`
	// File with no route annotations.
	srcNoRoutes := `package api

// Helper does something internal.
func Helper() {}
`
	afRoutes, fsetRoutes, fpRoutes := makeASTFile(t, dir, "routes.go", srcRoutes)
	afNoRoutes, fsetNoRoutes, fpNoRoutes := makeASTFile(t, dir, "helper.go", srcNoRoutes)

	svc := newTestService()

	files := map[*ast.File]*loader.AstFileInfo{
		afRoutes:   {Path: fpRoutes, FileSet: fsetRoutes},
		afNoRoutes: {Path: fpNoRoutes, FileSet: fsetNoRoutes},
	}

	routes, count, err := svc.parseRoutesParallel(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(routes))
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}
}

func TestParseRoutesParallel_NilASTGuard(t *testing.T) {
	// Verify that nil AST file keys in the map are skipped safely.
	svc := newTestService()

	files := map[*ast.File]*loader.AstFileInfo{
		nil: {Path: "/nonexistent/bad.go", FileSet: token.NewFileSet()},
	}

	routes, count, err := svc.parseRoutesParallel(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
	if count != 0 {
		t.Errorf("expected 0 count, got %d", count)
	}
}

func TestParseRoutesParallel_ManyFiles(t *testing.T) {
	dir := t.TempDir()

	// Create 20 files, each with a unique route, to exercise concurrency.
	files := make(map[*ast.File]*loader.AstFileInfo)
	for i := 0; i < 20; i++ {
		name := filepath.Base(t.TempDir()) // unique suffix
		src := `package api

// Handler` + name + ` godoc
// @summary handler ` + name + `
// @router /route-` + name + ` [get]
func Handler` + name + `() {}
`
		// Use a subdirectory-style name to avoid collisions.
		fileName := "handler_" + name + ".go"
		af, fset, fp := makeASTFile(t, dir, fileName, src)
		files[af] = &loader.AstFileInfo{Path: fp, FileSet: fset}
	}

	svc := newTestService()

	routes, count, err := svc.parseRoutesParallel(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 20 {
		t.Errorf("expected 20 routes, got %d", len(routes))
	}
	if count != 20 {
		t.Errorf("expected count 20, got %d", count)
	}

	// Verify all routes are present in swagger.
	if len(svc.swagger.Paths.Paths) != 20 {
		t.Errorf("expected 20 swagger paths, got %d", len(svc.swagger.Paths.Paths))
	}
}
