package orchestrator

import (
	"testing"

	routedomain "github.com/griffnb/core-swag/internal/parser/route/domain"
)

func TestCollectReferencedTypes_EmptyRoutes(t *testing.T) {
	refs := CollectReferencedTypes(nil)
	if len(refs) != 0 {
		t.Fatalf("expected empty map for nil routes, got %d entries", len(refs))
	}

	refs = CollectReferencedTypes([]*routedomain.Route{})
	if len(refs) != 0 {
		t.Fatalf("expected empty map for empty routes, got %d entries", len(refs))
	}
}

func TestCollectReferencedTypes_NilRouteInSlice(t *testing.T) {
	refs := CollectReferencedTypes([]*routedomain.Route{nil, nil})
	if len(refs) != 0 {
		t.Fatalf("expected empty map for nil route entries, got %d entries", len(refs))
	}
}

func TestCollectReferencedTypes_BodyParamRef(t *testing.T) {
	routes := []*routedomain.Route{
		{
			Parameters: []routedomain.Parameter{
				{
					Name: "body",
					In:   "body",
					Schema: &routedomain.Schema{
						Ref: "#/definitions/account.CreateRequest",
					},
				},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	if !refs["account.CreateRequest"] {
		t.Fatalf("expected account.CreateRequest in refs, got %v", refs)
	}
	if len(refs) != 1 {
		t.Fatalf("expected exactly 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestCollectReferencedTypes_ResponseRef(t *testing.T) {
	routes := []*routedomain.Route{
		{
			Responses: map[int]routedomain.Response{
				200: {
					Schema: &routedomain.Schema{
						Ref: "#/definitions/user.User",
					},
				},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	if !refs["user.User"] {
		t.Fatalf("expected user.User in refs, got %v", refs)
	}
	if len(refs) != 1 {
		t.Fatalf("expected exactly 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestCollectReferencedTypes_ArrayItemsRef(t *testing.T) {
	routes := []*routedomain.Route{
		{
			Responses: map[int]routedomain.Response{
				200: {
					Schema: &routedomain.Schema{
						Type: "array",
						Items: &routedomain.Schema{
							Ref: "#/definitions/order.OrderLine",
						},
					},
				},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	if !refs["order.OrderLine"] {
		t.Fatalf("expected order.OrderLine in refs, got %v", refs)
	}
	if len(refs) != 1 {
		t.Fatalf("expected exactly 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestCollectReferencedTypes_AllOfRefs(t *testing.T) {
	routes := []*routedomain.Route{
		{
			Responses: map[int]routedomain.Response{
				200: {
					Schema: &routedomain.Schema{
						AllOf: []*routedomain.Schema{
							{Ref: "#/definitions/base.BaseModel"},
							{Ref: "#/definitions/account.Account"},
						},
					},
				},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	if !refs["base.BaseModel"] {
		t.Fatalf("expected base.BaseModel in refs, got %v", refs)
	}
	if !refs["account.Account"] {
		t.Fatalf("expected account.Account in refs, got %v", refs)
	}
	if len(refs) != 2 {
		t.Fatalf("expected exactly 2 refs, got %d: %v", len(refs), refs)
	}
}

func TestCollectReferencedTypes_PropertiesRefs(t *testing.T) {
	routes := []*routedomain.Route{
		{
			Parameters: []routedomain.Parameter{
				{
					Name: "body",
					In:   "body",
					Schema: &routedomain.Schema{
						Type: "object",
						Properties: map[string]*routedomain.Schema{
							"billing": {Ref: "#/definitions/billing.Address"},
							"shipping": {Ref: "#/definitions/shipping.Address"},
						},
					},
				},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	if !refs["billing.Address"] {
		t.Fatalf("expected billing.Address in refs, got %v", refs)
	}
	if !refs["shipping.Address"] {
		t.Fatalf("expected shipping.Address in refs, got %v", refs)
	}
	if len(refs) != 2 {
		t.Fatalf("expected exactly 2 refs, got %d: %v", len(refs), refs)
	}
}

func TestCollectReferencedTypes_Deduplication(t *testing.T) {
	// Same type referenced from two different routes and locations.
	routes := []*routedomain.Route{
		{
			Parameters: []routedomain.Parameter{
				{
					Name:   "body",
					In:     "body",
					Schema: &routedomain.Schema{Ref: "#/definitions/shared.ErrorResponse"},
				},
			},
			Responses: map[int]routedomain.Response{
				400: {Schema: &routedomain.Schema{Ref: "#/definitions/shared.ErrorResponse"}},
			},
		},
		{
			Responses: map[int]routedomain.Response{
				500: {Schema: &routedomain.Schema{Ref: "#/definitions/shared.ErrorResponse"}},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	if !refs["shared.ErrorResponse"] {
		t.Fatalf("expected shared.ErrorResponse in refs, got %v", refs)
	}
	if len(refs) != 1 {
		t.Fatalf("expected exactly 1 ref after dedup, got %d: %v", len(refs), refs)
	}
}

func TestCollectReferencedTypes_FullPathRef(t *testing.T) {
	routes := []*routedomain.Route{
		{
			Responses: map[int]routedomain.Response{
				200: {
					Schema: &routedomain.Schema{
						Ref: "#/definitions/account.AccountPublic",
					},
				},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	if !refs["account.AccountPublic"] {
		t.Fatalf("expected account.AccountPublic in refs, got %v", refs)
	}
}

func TestCollectReferencedTypes_PrimitiveOnly(t *testing.T) {
	routes := []*routedomain.Route{
		{
			Parameters: []routedomain.Parameter{
				{Name: "id", In: "path", Type: "string"},
				{Name: "limit", In: "query", Type: "integer"},
			},
			Responses: map[int]routedomain.Response{
				200: {
					Schema: &routedomain.Schema{Type: "string"},
				},
				204: {Description: "No Content"},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	if len(refs) != 0 {
		t.Fatalf("expected empty map for primitive-only routes, got %d: %v", len(refs), refs)
	}
}

func TestCollectReferencedTypes_NilSchemaFields(t *testing.T) {
	// Every schema pointer is nil -- must not panic.
	routes := []*routedomain.Route{
		{
			Parameters: []routedomain.Parameter{
				{Name: "x", Schema: nil},
				{Name: "y"}, // Schema zero value
			},
			Responses: map[int]routedomain.Response{
				200: {Schema: nil},
				201: {},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	if len(refs) != 0 {
		t.Fatalf("expected empty map when all schemas nil, got %d: %v", len(refs), refs)
	}
}

func TestCollectReferencedTypes_DeeplyNested(t *testing.T) {
	// Array of objects where each object property itself has an Items array with a ref.
	routes := []*routedomain.Route{
		{
			Responses: map[int]routedomain.Response{
				200: {
					Schema: &routedomain.Schema{
						Type: "array",
						Items: &routedomain.Schema{
							Type: "object",
							Properties: map[string]*routedomain.Schema{
								"tags": {
									Type: "array",
									Items: &routedomain.Schema{
										Ref: "#/definitions/tag.Tag",
									},
								},
								"category": {
									Ref: "#/definitions/catalog.Category",
								},
							},
						},
					},
				},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	if !refs["tag.Tag"] {
		t.Fatalf("expected tag.Tag in refs, got %v", refs)
	}
	if !refs["catalog.Category"] {
		t.Fatalf("expected catalog.Category in refs, got %v", refs)
	}
	if len(refs) != 2 {
		t.Fatalf("expected exactly 2 refs, got %d: %v", len(refs), refs)
	}
}

func TestCollectReferencedTypes_MixedParamsAndResponses(t *testing.T) {
	routes := []*routedomain.Route{
		{
			Parameters: []routedomain.Parameter{
				{
					Name: "body",
					In:   "body",
					Schema: &routedomain.Schema{
						Ref: "#/definitions/invoice.CreateInput",
					},
				},
			},
			Responses: map[int]routedomain.Response{
				200: {
					Schema: &routedomain.Schema{
						Ref: "#/definitions/invoice.Invoice",
					},
				},
				400: {
					Schema: &routedomain.Schema{
						Ref: "#/definitions/shared.ErrorResponse",
					},
				},
			},
		},
	}

	refs := CollectReferencedTypes(routes)

	expected := []string{"invoice.CreateInput", "invoice.Invoice", "shared.ErrorResponse"}
	for _, name := range expected {
		if !refs[name] {
			t.Errorf("expected %s in refs, got %v", name, refs)
		}
	}
	if len(refs) != len(expected) {
		t.Fatalf("expected %d refs, got %d: %v", len(expected), len(refs), refs)
	}
}
