package testing_test

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/griffnb/core-swag/internal/loader"
	"github.com/griffnb/core-swag/internal/orchestrator"
	"github.com/griffnb/core/lib/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealProjectIntegration(t *testing.T) {
	mainAPIFile := "main.go"

	// Create orchestrator with config
	config := &orchestrator.Config{
		ParseDependency: loader.ParseFlag(1),
		ParseGoList:     true,
		ParseInternal:   true, // Include internal packages for this test
	}
	service := orchestrator.New(config)

	// Parse using new API
	swagger, err := service.Parse([]string{
		"/Users/griffnb/projects/Crowdshield/atlas-go/cmd/server",
		"/Users/griffnb/projects/Crowdshield/atlas-go/internal/controllers",
		"/Users/griffnb/projects/Crowdshield/atlas-go/internal/models",
	}, mainAPIFile, 100)
	require.NoError(t, err, "Failed to parse API")

	actualJSON, err := json.MarshalIndent(swagger, "", "  ")
	require.NoError(t, err, "Failed to marshal swagger to JSON")

	err = os.WriteFile("real_actual_output.json", actualJSON, 0o644)
	require.NoError(t, err, "Failed to write swagger JSON to file")
}

func TestCoreModelsIntegration(t *testing.T) {
	log.Debugf("Starting TestCoreModelsIntegration")

	searchDir := "testdata/core_models"
	mainAPIFile := "main.go"

	// Create orchestrator with config
	config := &orchestrator.Config{
		ParseDependency: loader.ParseFlag(1),
	}
	service := orchestrator.New(config)

	// Parse using new API
	swagger, err := service.Parse([]string{searchDir}, mainAPIFile, 100)
	require.NoError(t, err, "Failed to parse API")

	// Debug: Print all definitions
	t.Logf("Total definitions generated: %d", len(swagger.Definitions))
	for name := range swagger.Definitions {
		t.Logf("  - %s", name)
	}

	// Test that base schemas exist
	t.Run("Base schemas should exist", func(t *testing.T) {
		assert.Contains(t, swagger.Definitions, "account.Account", "account.Account definition should exist")
		assert.Contains(t, swagger.Definitions, "account.AccountJoined", "account.AccountJoined definition should exist")
		assert.Contains(t, swagger.Definitions, "billing_plan.BillingPlanJoined", "billing_plan.BillingPlanJoined definition should exist")
	})

	// Test that Public variant schemas exist
	t.Run("Public variant schemas should exist", func(t *testing.T) {
		assert.Contains(t, swagger.Definitions, "account.AccountPublic", "account.AccountPublic definition should exist")
		assert.Contains(t, swagger.Definitions, "account.AccountJoinedPublic", "account.AccountJoinedPublic definition should exist")
		assert.Contains(
			t,
			swagger.Definitions,
			"billing_plan.BillingPlanJoinedPublic",
			"billing_plan.BillingPlanJoinedPublic definition should exist",
		)
	})

	// Test field properties in base Account schema
	t.Run("Base Account schema should have correct fields", func(t *testing.T) {
		accountSchema := swagger.Definitions["account.Account"]
		require.NotNil(t, accountSchema, "account.Account schema should exist")

		props := accountSchema.Properties

		// Check that all fields exist (including non-public ones)
		assert.Contains(t, props, "first_name", "Should have first_name field")
		assert.Contains(t, props, "last_name", "Should have last_name field")
		assert.Contains(t, props, "email", "Should have email field")
		assert.Contains(t, props, "hashed_password", "Should have hashed_password field (private)")
		assert.Contains(t, props, "properties", "Should have properties field (private struct)")
		assert.Contains(t, props, "signup_properties", "Should have signup_properties field (private struct)")

		// Log all properties for debugging
		t.Logf("Base Account properties (%d total):", len(props))
		for propName := range props {
			t.Logf("  - %s", propName)
		}
	})

	// Test enum values on Account fields
	t.Run("Account role field should have enum values from IntConstantField[constants.Role]", func(t *testing.T) {
		accountSchema := swagger.Definitions["account.Account"]
		require.NotNil(t, accountSchema, "account.Account schema should exist")

		roleSchema, hasRole := accountSchema.Properties["role"]
		require.True(t, hasRole, "Account should have role field")

		// role is *fields.IntConstantField[constants.Role] which should resolve to integer with enum values
		assert.Contains(t, roleSchema.Type, "integer", "role should be integer type")
		require.NotEmpty(t, roleSchema.Enum, "role should have enum values from constants.Role")

		// Expected enum values: ROLE_UNAUTHORIZED=-1, ROLE_ANY_AUTHORIZED=0, ROLE_USER=1,
		// ROLE_ORG_ADMIN=40, ROLE_ORG_OWNER=50, ROLE_ADMIN=100
		// Note: ROLE_READ_ADMIN=90 has no explicit Role type, so it should be excluded
		expectedEnums := []int{-1, 0, 1, 40, 50, 100}
		actualEnums := make([]int, 0, len(roleSchema.Enum))
		for _, v := range roleSchema.Enum {
			switch val := v.(type) {
			case int:
				actualEnums = append(actualEnums, val)
			case float64:
				actualEnums = append(actualEnums, int(val))
			}
		}
		sort.Ints(expectedEnums)
		sort.Ints(actualEnums)
		assert.Equal(t, expectedEnums, actualEnums, "role enum values should match constants.Role values")
	})

	t.Run("Account config_key field should have enum values from StringConstantField[constants.GlobalConfigKey]", func(t *testing.T) {
		accountSchema := swagger.Definitions["account.Account"]
		require.NotNil(t, accountSchema, "account.Account schema should exist")

		configKeySchema, hasConfigKey := accountSchema.Properties["config_key"]
		require.True(t, hasConfigKey, "Account should have config_key field")

		// config_key is *fields.StringConstantField[constants.GlobalConfigKey]
		assert.Contains(t, configKeySchema.Type, "string", "config_key should be string type")
		require.NotEmpty(t, configKeySchema.Enum, "config_key should have enum values from constants.GlobalConfigKey")

		expectedEnums := []string{"allow_self_signed_certs", "api_rate_limit_enabled"}
		actualEnums := make([]string, 0, len(configKeySchema.Enum))
		for _, v := range configKeySchema.Enum {
			if s, ok := v.(string); ok {
				actualEnums = append(actualEnums, s)
			}
		}
		sort.Strings(expectedEnums)
		sort.Strings(actualEnums)
		assert.Equal(t, expectedEnums, actualEnums, "config_key enum values should match constants.GlobalConfigKey values")
	})

	t.Run("Properties nj_dl_classification should have enum values", func(t *testing.T) {
		propsSchema := swagger.Definitions["account.Properties"]
		require.NotNil(t, propsSchema, "account.Properties schema should exist")

		njdlSchema, hasNJDL := propsSchema.Properties["nj_dl_classification"]
		require.True(t, hasNJDL, "Properties should have nj_dl_classification field")

		assert.Contains(t, njdlSchema.Type, "integer", "nj_dl_classification should be integer type")
		require.NotEmpty(t, njdlSchema.Enum, "nj_dl_classification should have enum values from constants.NJDLClassification")

		// Expected: NJ_DL_LAW_ENFORCEMENT_OFFICER=1 through NJ_DL_FAMILY_MEMBER=5
		expectedEnums := []int{1, 2, 3, 4, 5}
		actualEnums := make([]int, 0, len(njdlSchema.Enum))
		for _, v := range njdlSchema.Enum {
			switch val := v.(type) {
			case int:
				actualEnums = append(actualEnums, val)
			case float64:
				actualEnums = append(actualEnums, int(val))
			}
		}
		sort.Ints(actualEnums)
		assert.Equal(t, expectedEnums, actualEnums, "nj_dl_classification enum values should be 1-5")
	})

	t.Run("Account status field should have enum values from IntConstantField[constants.Status]", func(t *testing.T) {
		accountSchema := swagger.Definitions["account.Account"]
		require.NotNil(t, accountSchema, "account.Account schema should exist")

		statusSchema, hasStatus := accountSchema.Properties["status"]
		require.True(t, hasStatus, "Account should have status field")

		assert.Contains(t, statusSchema.Type, "integer", "status should be integer type")
		require.NotEmpty(t, statusSchema.Enum, "status should have enum values from constants.Status")

		// Expected: STATUS_ACTIVE=100, STATUS_DISABLED=200, STATUS_DELETED=300
		expectedEnums := []int{100, 200, 300}
		actualEnums := make([]int, 0, len(statusSchema.Enum))
		for _, v := range statusSchema.Enum {
			switch val := v.(type) {
			case int:
				actualEnums = append(actualEnums, val)
			case float64:
				actualEnums = append(actualEnums, int(val))
			}
		}
		sort.Ints(actualEnums)
		assert.Equal(t, expectedEnums, actualEnums, "status enum values should match constants.Status values")
	})

	// Test field properties in Public Account schema
	t.Run("Public Account schema should filter private fields", func(t *testing.T) {
		accountPublicSchema := swagger.Definitions["account.AccountPublic"]
		require.NotNil(t, accountPublicSchema, "account.AccountPublic schema should exist")

		props := accountPublicSchema.Properties

		// Check that public:"view" or public:"edit" fields exist
		assert.Contains(t, props, "first_name", "Should have first_name field (public:edit)")
		assert.Contains(t, props, "email", "Should have email field (public:edit)")
		assert.Contains(t, props, "external_id", "Should have external_id field (public:view)")

		// Check that private fields are excluded
		assert.NotContains(t, props, "hashed_password", "Should NOT have hashed_password field (no public tag)")
		assert.NotContains(t, props, "properties", "Should NOT have properties field (no public tag)")
		assert.NotContains(t, props, "signup_properties", "Should NOT have signup_properties field (no public tag)")

		// Log all properties for debugging
		t.Logf("Public Account properties (%d total):", len(props))
		for propName := range props {
			t.Logf("  - %s", propName)
		}
	})

	// Test operations and their schema references
	t.Run("Operations should reference correct schemas", func(t *testing.T) {
		// Test /auth/me endpoint (has @Public annotation)
		authMePath := swagger.Paths.Paths["/auth/me"]
		require.NotNil(t, authMePath, "/auth/me path should exist")

		meOperation := authMePath.Get
		require.NotNil(t, meOperation, "/auth/me GET operation should exist")

		// Check 200 response
		response200 := meOperation.Responses.StatusCodeResponses[200]
		require.NotNil(t, response200, "/auth/me should have 200 response")
		require.NotNil(t, response200.Schema, "Response schema should not be nil")

		// The response wraps data in response.SuccessResponse{data=account.AccountJoined}
		// Because of @Public annotation, data field should reference account.AccountJoinedPublic
		t.Logf("/auth/me 200 response schema: %+v", response200.Schema)

		// The schema should be a composed schema (AllOf)
		require.NotNil(t, response200.Schema.AllOf, "Response should use AllOf for combined schema")
		require.Len(t, response200.Schema.AllOf, 2, "AllOf should have 2 parts")

		// First part references response.SuccessResponse (outer envelope)
		assert.Equal(t, "#/definitions/response.SuccessResponse", response200.Schema.AllOf[0].Ref.String(),
			"First part should reference response.SuccessResponse")

		// Second part has data property
		require.NotNil(t, response200.Schema.AllOf[1].Properties, "Second part should have properties")
		dataSchema, hasData := response200.Schema.AllOf[1].Properties["data"]
		require.True(t, hasData, "Should have data property")

		// Data property should reference account.AccountWithFeaturesPublic (not base AccountWithFeatures)
		assert.Equal(t, "#/definitions/account.AccountWithFeaturesPublic", dataSchema.Ref.String(),
			"@Public endpoint should reference AccountWithFeaturesPublic")

		// Test /admin/testUser endpoint (no @Public annotation)
		// This should use base schemas, not Public variants
		adminTestUserPath := swagger.Paths.Paths["/admin/testUser"]
		require.NotNil(t, adminTestUserPath, "/admin/testUser path should exist")

		createTestOp := adminTestUserPath.Post
		require.NotNil(t, createTestOp, "/admin/testUser POST operation should exist")

		// Check 200 response
		createResponse200 := createTestOp.Responses.StatusCodeResponses[200]
		require.NotNil(t, createResponse200, "/admin/testUser should have 200 response")
		require.NotNil(t, createResponse200.Schema, "Response schema should not be nil")

		t.Logf("/admin/testUser 200 response schema: %+v", createResponse200.Schema)

		// This endpoint doesn't have @Public, so should use base Account schema
		require.NotNil(t, createResponse200.Schema.AllOf, "Response should use AllOf")
		require.Len(t, createResponse200.Schema.AllOf, 2, "AllOf should have 2 parts")

		createDataSchema, hasCreateData := createResponse200.Schema.AllOf[1].Properties["data"]
		require.True(t, hasCreateData, "Should have data property")

		// Without @Public, should reference base account.Account (not AccountPublic)
		assert.Equal(t, "#/definitions/account.Account", createDataSchema.Ref.String(),
			"Non-public endpoint should reference base Account schema")

		// Test /api/account/{id} endpoint (no @Public annotation)
		apiAccountPath := swagger.Paths.Paths["/api/account/{id}"]
		require.NotNil(t, apiAccountPath, "/api/account/{id} path should exist")

		apiAccountOp := apiAccountPath.Get
		require.NotNil(t, apiAccountOp, "/api/account/{id} GET operation should exist")

		// Check 200 response references APIResponse with BillingPlanJoined
		apiAccountResponse200 := apiAccountOp.Responses.StatusCodeResponses[200]
		require.NotNil(t, apiAccountResponse200, "/api/account/{id} should have 200 response")
		require.NotNil(t, apiAccountResponse200.Schema, "Response schema should not be nil")

		t.Logf("/api/account/{id} 200 response schema: %+v", apiAccountResponse200.Schema)
	})

	// Write actual output to a file for comparison
	t.Run("Generate actual output", func(t *testing.T) {
		actualJSON, err := json.MarshalIndent(swagger, "", "  ")
		require.NoError(t, err, "Failed to marshal swagger to JSON")

		err = os.WriteFile("actual_output.json", actualJSON, 0o644)
		require.NoError(t, err, "Failed to write actual output")

		t.Logf("Actual swagger output written to actual_output.json")
		t.Logf("Total paths: %d", len(swagger.Paths.Paths))
		for path := range swagger.Paths.Paths {
			t.Logf("  - %s", path)
		}
	})
}

func TestAccountJoinedSchema(t *testing.T) {
	searchDir := "testdata/core_models"
	mainAPIFile := "main.go"

	// Create orchestrator with config
	config := &orchestrator.Config{
		ParseDependency: loader.ParseFlag(1),
	}
	service := orchestrator.New(config)

	// Parse using new API
	swagger, err := service.Parse([]string{searchDir}, mainAPIFile, 100)
	require.NoError(t, err)

	t.Run("AccountJoined should include JoinData fields", func(t *testing.T) {
		schema := swagger.Definitions["account.AccountJoined"]
		require.NotNil(t, schema, "account.AccountJoined should exist")

		props := schema.Properties

		// JoinData fields (should all be present in base schema)
		assert.Contains(t, props, "name", "Should have name from JoinData")
		assert.Contains(t, props, "organization_name", "Should have organization_name from JoinData")
		assert.Contains(t, props, "created_by_name", "Should have created_by_name from JoinData")
		assert.Contains(t, props, "updated_by_name", "Should have updated_by_name from JoinData")

		// DBColumns fields
		assert.Contains(t, props, "first_name", "Should have first_name from DBColumns")
		assert.Contains(t, props, "email", "Should have email from DBColumns")

		t.Logf("AccountJoined has %d properties", len(props))
	})

	t.Run("AccountJoinedPublic should filter private fields but keep public JoinData", func(t *testing.T) {
		schema := swagger.Definitions["account.AccountJoinedPublic"]
		require.NotNil(t, schema, "account.AccountJoinedPublic should exist")

		props := schema.Properties

		// JoinData fields - these don't have public tags, so check if they're included
		// Based on the struct, JoinData fields don't have public tags, so they might be excluded
		t.Logf("AccountJoinedPublic has %d properties", len(props))
		for propName := range props {
			t.Logf("  - %s", propName)
		}

		// Public fields should exist
		assert.Contains(t, props, "first_name", "Should have first_name (public:edit)")
		assert.Contains(t, props, "external_id", "Should have external_id (public:view)")

		// Private fields should not exist
		assert.NotContains(t, props, "hashed_password", "Should NOT have hashed_password")
	})
}

func TestBillingPlanSchema(t *testing.T) {
	searchDir := "testdata/core_models"
	mainAPIFile := "main.go"

	// Create orchestrator with config
	config := &orchestrator.Config{
		ParseDependency: loader.ParseFlag(1),
	}
	service := orchestrator.New(config)

	// Parse using new API
	swagger, err := service.Parse([]string{searchDir}, mainAPIFile, 100)
	require.NoError(t, err)

	t.Run("BillingPlanJoined should have nested StructField types", func(t *testing.T) {
		schema := swagger.Definitions["billing_plan.BillingPlanJoined"]
		require.NotNil(t, schema, "billing_plan.BillingPlanJoined should exist")

		props := schema.Properties

		// Check for StructField fields
		assert.Contains(t, props, "feature_set", "Should have feature_set (StructField)")
		assert.Contains(t, props, "properties", "Should have properties (StructField)")

		// Check basic fields
		assert.Contains(t, props, "name", "Should have name")
		assert.Contains(t, props, "description", "Should have description")

		t.Logf("BillingPlanJoined has %d properties", len(props))
		for propName := range props {
			t.Logf("  - %s", propName)
		}
	})

	t.Run("BillingPlanJoinedPublic should filter private fields", func(t *testing.T) {
		schema := swagger.Definitions["billing_plan.BillingPlanJoinedPublic"]
		require.NotNil(t, schema, "billing_plan.BillingPlanJoinedPublic should exist")

		props := schema.Properties

		t.Logf("BillingPlanJoinedPublic has %d properties", len(props))
		for propName := range props {
			t.Logf("  - %s", propName)
		}

		// Should have public fields
		assert.Contains(t, props, "name", "Should have name (public:view)")
		assert.Contains(t, props, "description", "Should have description (public:view)")

		// feature_set has public:"view" so should be included
		assert.Contains(t, props, "feature_set", "Should have feature_set (public:view)")
	})
}
