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

	// Test enum fields use $ref to enum definitions instead of inlining values
	t.Run("Account role field should $ref constants.Role definition", func(t *testing.T) {
		accountSchema := swagger.Definitions["account.Account"]
		require.NotNil(t, accountSchema, "account.Account schema should exist")

		roleSchema, hasRole := accountSchema.Properties["role"]
		require.True(t, hasRole, "Account should have role field")

		// role should be a $ref to constants.Role definition
		assert.Equal(t, "#/definitions/constants.Role", roleSchema.Ref.String(), "role should $ref constants.Role")

		// Verify the definition exists with enum values and x-enum-varnames
		roleDef, hasDef := swagger.Definitions["constants.Role"]
		require.True(t, hasDef, "constants.Role definition should exist")
		assert.Contains(t, roleDef.Type, "integer", "constants.Role should be integer type")
		require.NotEmpty(t, roleDef.Enum, "constants.Role should have enum values")

		expectedEnums := []int{-1, 0, 1, 40, 50, 100}
		actualEnums := make([]int, 0, len(roleDef.Enum))
		for _, v := range roleDef.Enum {
			switch val := v.(type) {
			case int:
				actualEnums = append(actualEnums, val)
			case float64:
				actualEnums = append(actualEnums, int(val))
			}
		}
		sort.Ints(expectedEnums)
		sort.Ints(actualEnums)
		assert.Equal(t, expectedEnums, actualEnums, "constants.Role enum values should match")

		// Verify x-enum-varnames extension
		varNames, hasVarNames := roleDef.Extensions["x-enum-varnames"]
		require.True(t, hasVarNames, "constants.Role should have x-enum-varnames")
		require.NotNil(t, varNames, "x-enum-varnames should not be nil")
	})

	t.Run("Account config_key field should $ref constants.GlobalConfigKey definition", func(t *testing.T) {
		accountSchema := swagger.Definitions["account.Account"]
		require.NotNil(t, accountSchema, "account.Account schema should exist")

		configKeySchema, hasConfigKey := accountSchema.Properties["config_key"]
		require.True(t, hasConfigKey, "Account should have config_key field")

		// config_key should be a $ref to constants.GlobalConfigKey definition
		assert.Equal(t, "#/definitions/constants.GlobalConfigKey", configKeySchema.Ref.String(), "config_key should $ref constants.GlobalConfigKey")

		// Verify the definition exists with enum values and x-enum-varnames
		configDef, hasDef := swagger.Definitions["constants.GlobalConfigKey"]
		require.True(t, hasDef, "constants.GlobalConfigKey definition should exist")
		assert.Contains(t, configDef.Type, "string", "constants.GlobalConfigKey should be string type")
		require.NotEmpty(t, configDef.Enum, "constants.GlobalConfigKey should have enum values")

		expectedEnums := []string{"allow_self_signed_certs", "api_rate_limit_enabled"}
		actualEnums := make([]string, 0, len(configDef.Enum))
		for _, v := range configDef.Enum {
			if s, ok := v.(string); ok {
				actualEnums = append(actualEnums, s)
			}
		}
		sort.Strings(expectedEnums)
		sort.Strings(actualEnums)
		assert.Equal(t, expectedEnums, actualEnums, "constants.GlobalConfigKey enum values should match")

		varNames, hasVarNames := configDef.Extensions["x-enum-varnames"]
		require.True(t, hasVarNames, "constants.GlobalConfigKey should have x-enum-varnames")
		require.NotNil(t, varNames, "x-enum-varnames should not be nil")
	})

	t.Run("Properties nj_dl_classification should $ref constants.NJDLClassification definition", func(t *testing.T) {
		propsSchema := swagger.Definitions["account.Properties"]
		require.NotNil(t, propsSchema, "account.Properties schema should exist")

		njdlSchema, hasNJDL := propsSchema.Properties["nj_dl_classification"]
		require.True(t, hasNJDL, "Properties should have nj_dl_classification field")

		// nj_dl_classification should be a $ref to constants.NJDLClassification definition
		assert.Equal(t, "#/definitions/constants.NJDLClassification", njdlSchema.Ref.String(), "nj_dl_classification should $ref constants.NJDLClassification")

		// Verify the definition exists with enum values
		njdlDef, hasDef := swagger.Definitions["constants.NJDLClassification"]
		require.True(t, hasDef, "constants.NJDLClassification definition should exist")
		assert.Contains(t, njdlDef.Type, "integer", "constants.NJDLClassification should be integer type")

		expectedEnums := []int{1, 2, 3, 4, 5}
		actualEnums := make([]int, 0, len(njdlDef.Enum))
		for _, v := range njdlDef.Enum {
			switch val := v.(type) {
			case int:
				actualEnums = append(actualEnums, val)
			case float64:
				actualEnums = append(actualEnums, int(val))
			}
		}
		sort.Ints(actualEnums)
		assert.Equal(t, expectedEnums, actualEnums, "constants.NJDLClassification enum values should be 1-5")

		varNames, hasVarNames := njdlDef.Extensions["x-enum-varnames"]
		require.True(t, hasVarNames, "constants.NJDLClassification should have x-enum-varnames")
		require.NotNil(t, varNames, "x-enum-varnames should not be nil")
	})

	t.Run("Account status field should $ref constants.Status definition", func(t *testing.T) {
		accountSchema := swagger.Definitions["account.Account"]
		require.NotNil(t, accountSchema, "account.Account schema should exist")

		statusSchema, hasStatus := accountSchema.Properties["status"]
		require.True(t, hasStatus, "Account should have status field")

		// status should be a $ref to constants.Status definition
		assert.Equal(t, "#/definitions/constants.Status", statusSchema.Ref.String(), "status should $ref constants.Status")

		// Verify the definition exists with enum values
		statusDef, hasDef := swagger.Definitions["constants.Status"]
		require.True(t, hasDef, "constants.Status definition should exist")
		assert.Contains(t, statusDef.Type, "integer", "constants.Status should be integer type")

		expectedEnums := []int{100, 200, 300}
		actualEnums := make([]int, 0, len(statusDef.Enum))
		for _, v := range statusDef.Enum {
			switch val := v.(type) {
			case int:
				actualEnums = append(actualEnums, val)
			case float64:
				actualEnums = append(actualEnums, int(val))
			}
		}
		sort.Ints(actualEnums)
		assert.Equal(t, expectedEnums, actualEnums, "constants.Status enum values should match")

		varNames, hasVarNames := statusDef.Extensions["x-enum-varnames"]
		require.True(t, hasVarNames, "constants.Status should have x-enum-varnames")
		require.NotNil(t, varNames, "x-enum-varnames should not be nil")
	})

	// Test OrganizationType on AccountJoined (the specific bug reported)
	t.Run("AccountJoined organization_type should $ref constants.OrganizationType definition", func(t *testing.T) {
		ajSchema := swagger.Definitions["account.AccountJoined"]
		require.NotNil(t, ajSchema, "account.AccountJoined schema should exist")

		orgTypeSchema, hasOrgType := ajSchema.Properties["organization_type"]
		require.True(t, hasOrgType, "AccountJoined should have organization_type field")

		// organization_type should be a $ref to constants.OrganizationType definition
		assert.Equal(t, "#/definitions/constants.OrganizationType", orgTypeSchema.Ref.String(), "organization_type should $ref constants.OrganizationType")

		// Verify the definition exists with enum values and x-enum-varnames
		orgTypeDef, hasDef := swagger.Definitions["constants.OrganizationType"]
		require.True(t, hasDef, "constants.OrganizationType definition should exist")
		assert.Contains(t, orgTypeDef.Type, "integer", "constants.OrganizationType should be integer type")

		expectedEnums := []int{1, 2, 3, 4, 5, 6}
		actualEnums := make([]int, 0, len(orgTypeDef.Enum))
		for _, v := range orgTypeDef.Enum {
			switch val := v.(type) {
			case int:
				actualEnums = append(actualEnums, val)
			case float64:
				actualEnums = append(actualEnums, int(val))
			}
		}
		sort.Ints(actualEnums)
		assert.Equal(t, expectedEnums, actualEnums, "constants.OrganizationType enum values should be 1-6")

		varNames, hasVarNames := orgTypeDef.Extensions["x-enum-varnames"]
		require.True(t, hasVarNames, "constants.OrganizationType should have x-enum-varnames")
		require.NotNil(t, varNames, "x-enum-varnames should not be nil")

		// Verify the specific varnames
		if names, ok := varNames.([]string); ok {
			assert.Contains(t, names, "ORGANIZATION_TYPE_INTERNAL_TESTING")
			assert.Contains(t, names, "ORGANIZATION_TYPE_ENTERPRISE")
			assert.Contains(t, names, "ORGANIZATION_TYPE_B2C")
		}
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
		assert.Contains(t, props, "properties", "Should have properties field (has public:view tag)")
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
