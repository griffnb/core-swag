package route

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/griffnb/core-swag/internal/domain"
	"github.com/stretchr/testify/require"
)

// fakeTypeRegistry resolves a single qualified name to a fixed TypeSpecDef,
// simulating a NotUnique collision so we can verify $ref canonicalization without
// standing up a full registry.
type fakeTypeRegistry struct {
	match string
	def   *domain.TypeSpecDef
}

func (f *fakeTypeRegistry) FindTypeSpec(typeName string, _ *ast.File) *domain.TypeSpecDef {
	if strings.Contains(typeName, f.match) {
		return f.def
	}
	return nil
}

// TestParseRoutes_NotUniqueRefCanonicalized verifies that when a route references a
// type by its short package-qualified name (e.g. "atlasmail.InboxThread") and that
// type is NotUnique, the emitted $ref uses the canonical full-path name of the type
// the file actually imports — not the ambiguous short name.
func TestParseRoutes_NotUniqueRefCanonicalized(t *testing.T) {
	notUnique := &domain.TypeSpecDef{
		TypeSpec:  &ast.TypeSpec{Name: ast.NewIdent("InboxThread")},
		PkgPath:   "github.com/CrowdShield/atlas-go/internal/services/atlasmail",
		NotUnique: true,
		File:      &ast.File{Name: ast.NewIdent("atlasmail")},
	}
	canonical := notUnique.TypeName() // sanitized full path
	require.Equal(t, "github_com_CrowdShield_atlas-go_internal_services_atlasmail.InboxThread", canonical)

	service := NewService(nil, "")
	service.SetRegistry(&fakeTypeRegistry{match: "InboxThread", def: notUnique})

	src := `
package inbox_threads

import "github.com/CrowdShield/atlas-go/internal/services/atlasmail"

var _ = atlasmail.InboxThread{}

// adminIndex lists inbox threads
// @Summary List
// @Success 200 {object} response.SuccessResponse{data=[]atlasmail.InboxThread}
// @Router /admin/inbox_threads [get]
func adminIndex() {}
`
	fset := token.NewFileSet()
	astFile, err := goparser.ParseFile(fset, "admin.go", src, goparser.ParseComments)
	require.NoError(t, err)

	routes, err := service.ParseRoutes(astFile, "admin.go", fset)
	require.NoError(t, err)
	require.Len(t, routes, 1)

	resp, ok := routes[0].Responses[200]
	require.True(t, ok, "expected a 200 response")
	require.NotNil(t, resp.Schema)
	require.Len(t, resp.Schema.AllOf, 2, "expected SuccessResponse allOf wrapper")

	dataProp := resp.Schema.AllOf[1].Properties["data"]
	require.NotNil(t, dataProp, "expected data property")
	require.NotNil(t, dataProp.Items, "expected data array items")

	require.Equal(t, "#/definitions/"+canonical, dataProp.Items.Ref,
		"items $ref must be the canonical full-path name, not the ambiguous short name")
	require.Equal(t, notUnique.FullPath(), dataProp.Items.TypePath)
}
