package base

import (
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
)

func TestParseGeneralInfo(t *testing.T) {
	t.Parallel()

	t.Run("parse title, version, and description", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@title Test API",
			"@version 1.0.0",
			"@description This is a test API",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.Equal(t, "Test API", swagger.Info.Title)
		assert.Equal(t, "1.0.0", swagger.Info.Version)
		assert.Equal(t, "This is a test API", swagger.Info.Description)
	})

	t.Run("parse multiline description", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@description Line 1",
			"@description Line 2",
			"@description Line 3",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.Equal(t, "Line 1\nLine 2\nLine 3", swagger.Info.Description)
	})

	t.Run("parse termsOfService", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@termsOfService http://example.com/terms",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.Equal(t, "http://example.com/terms", swagger.Info.TermsOfService)
	})
}

func TestParseContactInfo(t *testing.T) {
	t.Parallel()

	t.Run("parse contact name, email, and url", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{
					InfoProps: spec.InfoProps{
						Contact: &spec.ContactInfo{},
					},
				},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@contact.name API Support",
			"@contact.email support@example.com",
			"@contact.url http://www.example.com/support",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.Equal(t, "API Support", swagger.Info.Contact.Name)
		assert.Equal(t, "support@example.com", swagger.Info.Contact.Email)
		assert.Equal(t, "http://www.example.com/support", swagger.Info.Contact.URL)
	})
}

func TestParseLicenseInfo(t *testing.T) {
	t.Parallel()

	t.Run("parse license name and url", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@license.name Apache 2.0",
			"@license.url http://www.apache.org/licenses/LICENSE-2.0.html",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.NotNil(t, swagger.Info.License)
		assert.Equal(t, "Apache 2.0", swagger.Info.License.Name)
		assert.Equal(t, "http://www.apache.org/licenses/LICENSE-2.0.html", swagger.Info.License.URL)
	})
}

func TestParseTagInfo(t *testing.T) {
	t.Parallel()

	t.Run("parse single tag with description", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@tag.name users",
			"@tag.description User management endpoints",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(swagger.Tags))
		assert.Equal(t, "users", swagger.Tags[0].Name)
		assert.Equal(t, "User management endpoints", swagger.Tags[0].Description)
	})

	t.Run("parse tag with external docs", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@tag.name users",
			"@tag.docs.url http://example.com/docs",
			"@tag.docs.description External documentation",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(swagger.Tags))
		assert.NotNil(t, swagger.Tags[0].ExternalDocs)
		assert.Equal(t, "http://example.com/docs", swagger.Tags[0].ExternalDocs.URL)
		assert.Equal(t, "External documentation", swagger.Tags[0].ExternalDocs.Description)
	})

	t.Run("parse multiple tags", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@tag.name users",
			"@tag.description User endpoints",
			"@tag.name products",
			"@tag.description Product endpoints",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(swagger.Tags))
		assert.Equal(t, "users", swagger.Tags[0].Name)
		assert.Equal(t, "User endpoints", swagger.Tags[0].Description)
		assert.Equal(t, "products", swagger.Tags[1].Name)
		assert.Equal(t, "Product endpoints", swagger.Tags[1].Description)
	})
}

func TestParseServerInfo(t *testing.T) {
	t.Parallel()

	t.Run("parse host and basepath", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@host localhost:8080",
			"@basePath /api/v1",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.Equal(t, "localhost:8080", swagger.Host)
		assert.Equal(t, "/api/v1", swagger.BasePath)
	})

	t.Run("parse schemes", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@schemes http https",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(swagger.Schemes))
		assert.Equal(t, "http", swagger.Schemes[0])
		assert.Equal(t, "https", swagger.Schemes[1])
	})
}

func TestParseExternalDocs(t *testing.T) {
	t.Parallel()

	t.Run("parse external docs url and description", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info: &spec.Info{},
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@externalDocs.description Find out more",
			"@externalDocs.url http://swagger.io",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.NotNil(t, swagger.ExternalDocs)
		assert.Equal(t, "Find out more", swagger.ExternalDocs.Description)
		assert.Equal(t, "http://swagger.io", swagger.ExternalDocs.URL)
	})
}

func TestParseSecurityDefinitions(t *testing.T) {
	t.Parallel()

	t.Run("parse basic auth", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info:                &spec.Info{},
				SecurityDefinitions: make(map[string]*spec.SecurityScheme),
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@securitydefinitions.basic BasicAuth",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.NotNil(t, swagger.SecurityDefinitions["BasicAuth"])
		assert.Equal(t, "basic", swagger.SecurityDefinitions["BasicAuth"].Type)
	})

	t.Run("parse apikey security", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info:                &spec.Info{},
				SecurityDefinitions: make(map[string]*spec.SecurityScheme),
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@securitydefinitions.apikey ApiKeyAuth",
			"@in header",
			"@name X-API-Key",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.NotNil(t, swagger.SecurityDefinitions["ApiKeyAuth"])
		assert.Equal(t, "apiKey", swagger.SecurityDefinitions["ApiKeyAuth"].Type)
		assert.Equal(t, "header", swagger.SecurityDefinitions["ApiKeyAuth"].In)
		assert.Equal(t, "X-API-Key", swagger.SecurityDefinitions["ApiKeyAuth"].Name)
	})

	t.Run("parse oauth2 implicit", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info:                &spec.Info{},
				SecurityDefinitions: make(map[string]*spec.SecurityScheme),
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@securitydefinitions.oauth2.implicit OAuth2Implicit",
			"@authorizationUrl https://example.com/oauth/authorize",
			"@scope.write write access",
			"@scope.read read access",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.NotNil(t, swagger.SecurityDefinitions["OAuth2Implicit"])
		assert.Equal(t, "oauth2", swagger.SecurityDefinitions["OAuth2Implicit"].Type)
		assert.Equal(t, "implicit", swagger.SecurityDefinitions["OAuth2Implicit"].Flow)
		assert.Equal(t, "https://example.com/oauth/authorize", swagger.SecurityDefinitions["OAuth2Implicit"].AuthorizationURL)
	})

	t.Run("parse oauth2 password", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info:                &spec.Info{},
				SecurityDefinitions: make(map[string]*spec.SecurityScheme),
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@securitydefinitions.oauth2.password OAuth2Password",
			"@tokenUrl https://example.com/oauth/token",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.NotNil(t, swagger.SecurityDefinitions["OAuth2Password"])
		assert.Equal(t, "oauth2", swagger.SecurityDefinitions["OAuth2Password"].Type)
		assert.Equal(t, "password", swagger.SecurityDefinitions["OAuth2Password"].Flow)
		assert.Equal(t, "https://example.com/oauth/token", swagger.SecurityDefinitions["OAuth2Password"].TokenURL)
	})

	t.Run("parse oauth2 application", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info:                &spec.Info{},
				SecurityDefinitions: make(map[string]*spec.SecurityScheme),
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@securitydefinitions.oauth2.application OAuth2Application",
			"@tokenUrl https://example.com/oauth/token",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.NotNil(t, swagger.SecurityDefinitions["OAuth2Application"])
		assert.Equal(t, "oauth2", swagger.SecurityDefinitions["OAuth2Application"].Type)
		assert.Equal(t, "application", swagger.SecurityDefinitions["OAuth2Application"].Flow)
		assert.Equal(t, "https://example.com/oauth/token", swagger.SecurityDefinitions["OAuth2Application"].TokenURL)
	})

	t.Run("parse oauth2 accessCode", func(t *testing.T) {
		swagger := &spec.Swagger{
			SwaggerProps: spec.SwaggerProps{
				Info:                &spec.Info{},
				SecurityDefinitions: make(map[string]*spec.SecurityScheme),
			},
		}
		service := NewService(swagger)

		comments := []string{
			"@securitydefinitions.oauth2.accessCode OAuth2AccessCode",
			"@tokenUrl https://example.com/oauth/token",
			"@authorizationUrl https://example.com/oauth/authorize",
		}

		err := service.ParseGeneralInfo(comments)
		assert.NoError(t, err)
		assert.NotNil(t, swagger.SecurityDefinitions["OAuth2AccessCode"])
		assert.Equal(t, "oauth2", swagger.SecurityDefinitions["OAuth2AccessCode"].Type)
		assert.Equal(t, "accessCode", swagger.SecurityDefinitions["OAuth2AccessCode"].Flow)
		assert.Equal(t, "https://example.com/oauth/token", swagger.SecurityDefinitions["OAuth2AccessCode"].TokenURL)
		assert.Equal(t, "https://example.com/oauth/authorize", swagger.SecurityDefinitions["OAuth2AccessCode"].AuthorizationURL)
	})
}
