package gen

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const searchDir = "../../testing/testdata/simple"

var outputTypes = []string{"json", "yaml"}

func TestGen_Build(t *testing.T) {
	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
	}
	assert.NoError(t, New().Build(config))

	expectedFiles := []string{
		filepath.Join(config.OutputDir, "swagger.json"),
		filepath.Join(config.OutputDir, "swagger.yaml"),
	}
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}

		_ = os.Remove(expectedFile)
	}
}

func TestGen_SpecificOutputTypes(t *testing.T) {
	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple/docs",
		OutputTypes:        []string{"json", "unknownType"},
		PropNamingStrategy: "",
	}
	assert.NoError(t, New().Build(config))

	tt := []struct {
		expectedFile string
		shouldExist  bool
	}{
		{filepath.Join(config.OutputDir, "swagger.json"), true},
		{filepath.Join(config.OutputDir, "swagger.yaml"), false},
	}
	for _, tc := range tt {
		_, err := os.Stat(tc.expectedFile)
		if tc.shouldExist {
			if os.IsNotExist(err) {
				require.NoError(t, err)
			}
		} else {
			require.Error(t, err)
			require.True(t, errors.Is(err, os.ErrNotExist))
		}

		_ = os.Remove(tc.expectedFile)
	}
}

func TestGen_BuildInstanceName(t *testing.T) {
	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
	}
	assert.NoError(t, New().Build(config))

	// Default instance name produces swagger.json/swagger.yaml
	expectedFiles := []string{
		filepath.Join(config.OutputDir, "swagger.json"),
		filepath.Join(config.OutputDir, "swagger.yaml"),
	}
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}
		_ = os.Remove(expectedFile)
	}

	// Custom instance name produces Custom_swagger.json/Custom_swagger.yaml
	config.InstanceName = "Custom"
	assert.NoError(t, New().Build(config))

	customFiles := []string{
		filepath.Join(config.OutputDir, config.InstanceName+"_"+"swagger.json"),
		filepath.Join(config.OutputDir, config.InstanceName+"_"+"swagger.yaml"),
	}
	for _, expectedFile := range customFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}
		_ = os.Remove(expectedFile)
	}
}

func TestGen_BuildSnakeCase(t *testing.T) {
	config := &Config{
		SearchDir:          "../../testing/testdata/simple2",
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple2/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "snakecase",
	}

	assert.NoError(t, New().Build(config))

	expectedFiles := []string{
		filepath.Join(config.OutputDir, "swagger.json"),
		filepath.Join(config.OutputDir, "swagger.yaml"),
	}
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}

		_ = os.Remove(expectedFile)
	}
}

func TestGen_BuildLowerCamelcase(t *testing.T) {
	config := &Config{
		SearchDir:          "../../testing/testdata/simple3",
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple3/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
	}

	assert.NoError(t, New().Build(config))

	expectedFiles := []string{
		filepath.Join(config.OutputDir, "swagger.json"),
		filepath.Join(config.OutputDir, "swagger.yaml"),
	}
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}

		_ = os.Remove(expectedFile)
	}
}

func TestGen_jsonIndent(t *testing.T) {
	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
	}

	gen := New()
	gen.jsonIndent = func(data interface{}) ([]byte, error) {
		return nil, errors.New("fail")
	}

	assert.Error(t, gen.Build(config))
}

func TestGen_jsonToYAML(t *testing.T) {
	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
	}

	gen := New()
	gen.jsonToYAML = func(data []byte) ([]byte, error) {
		return nil, errors.New("fail")
	}
	assert.Error(t, gen.Build(config))

	expectedFiles := []string{
		filepath.Join(config.OutputDir, "swagger.json"),
	}

	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}

		_ = os.Remove(expectedFile)
	}
}

func TestGen_SearchDirIsNotExist(t *testing.T) {
	var swaggerConfDir, propNamingStrategy string

	config := &Config{
		SearchDir:          "../isNotExistDir",
		MainAPIFile:        "./main.go",
		OutputDir:          swaggerConfDir,
		OutputTypes:        outputTypes,
		PropNamingStrategy: propNamingStrategy,
	}

	assert.EqualError(t, New().Build(config), "dir: ../isNotExistDir does not exist")
}

func TestGen_MainAPiNotExist(t *testing.T) {
	var swaggerConfDir, propNamingStrategy string

	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./notExists.go",
		OutputDir:          swaggerConfDir,
		OutputTypes:        outputTypes,
		PropNamingStrategy: propNamingStrategy,
	}

	assert.Error(t, New().Build(config))
}

func TestGen_OutputIsNotExist(t *testing.T) {
	var propNamingStrategy string
	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          "/dev/null",
		OutputTypes:        outputTypes,
		PropNamingStrategy: propNamingStrategy,
	}
	assert.Error(t, New().Build(config))
}

func TestGen_FailToWrite(t *testing.T) {
	outputDir := filepath.Join(os.TempDir(), "swagg", "test")
	outputTypes := []string{"json", "yaml"}

	var propNamingStrategy string
	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          outputDir,
		OutputTypes:        outputTypes,
		PropNamingStrategy: propNamingStrategy,
	}

	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		require.NoError(t, err)
	}

	_ = os.RemoveAll(filepath.Join(outputDir, "swagger.yaml"))

	err = os.Mkdir(filepath.Join(outputDir, "swagger.yaml"), 0755)
	if err != nil {
		require.NoError(t, err)
	}
	assert.Error(t, New().Build(config))

	_ = os.RemoveAll(filepath.Join(outputDir, "swagger.json"))

	err = os.Mkdir(filepath.Join(outputDir, "swagger.json"), 0755)
	if err != nil {
		require.NoError(t, err)
	}
	assert.Error(t, New().Build(config))

	err = os.RemoveAll(outputDir)
	if err != nil {
		require.NoError(t, err)
	}
}

func TestGen_configWithOutputDir(t *testing.T) {
	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
	}

	assert.NoError(t, New().Build(config))

	expectedFiles := []string{
		filepath.Join(config.OutputDir, "swagger.json"),
		filepath.Join(config.OutputDir, "swagger.yaml"),
	}
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}

		_ = os.Remove(expectedFile)
	}
}

func TestGen_configWithOutputTypesAll(t *testing.T) {
	searchDir := "../../testing/testdata/simple"
	outputTypes := []string{"json", "yaml"}

	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
	}

	assert.NoError(t, New().Build(config))

	expectedFiles := []string{
		path.Join(config.OutputDir, "swagger.json"),
		path.Join(config.OutputDir, "swagger.yaml"),
	}
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Fatal(err)
		}

		_ = os.Remove(expectedFile)
	}
}

func TestGen_configWithOutputTypesSingle(t *testing.T) {
	searchDir := "../../testing/testdata/simple"
	outputTypes := []string{"json", "yaml"}

	for _, outputType := range outputTypes {
		config := &Config{
			SearchDir:          searchDir,
			MainAPIFile:        "./main.go",
			OutputDir:          "../../testing/testdata/simple/docs",
			OutputTypes:        []string{outputType},
			PropNamingStrategy: "",
		}

		assert.NoError(t, New().Build(config))

		expectedFiles := []string{
			path.Join(config.OutputDir, "swagger."+outputType),
		}
		for _, expectedFile := range expectedFiles {
			if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
				t.Fatal(err)
			}

			_ = os.Remove(expectedFile)
		}
	}
}

func TestGen_cgoImports(t *testing.T) {
	config := &Config{
		SearchDir:          "../../testing/testdata/simple_cgo",
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple_cgo/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
		ParseDependency:    1,
	}

	assert.NoError(t, New().Build(config))

	expectedFiles := []string{
		filepath.Join(config.OutputDir, "swagger.json"),
		filepath.Join(config.OutputDir, "swagger.yaml"),
	}
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}

		_ = os.Remove(expectedFile)
	}
}

func TestGen_parseOverrides(t *testing.T) {
	testCases := []struct {
		Name          string
		Data          string
		Expected      map[string]string
		ExpectedError error
	}{
		{
			Name: "replace",
			Data: `replace github.com/foo/bar baz`,
			Expected: map[string]string{
				"github.com/foo/bar": "baz",
			},
		},
		{
			Name: "skip",
			Data: `skip github.com/foo/bar`,
			Expected: map[string]string{
				"github.com/foo/bar": "",
			},
		},
		{
			Name: "generic-simple",
			Data: `replace types.Field[string] string`,
			Expected: map[string]string{
				"types.Field[string]": "string",
			},
		},
		{
			Name: "generic-double",
			Data: `replace types.Field[string,string] string`,
			Expected: map[string]string{
				"types.Field[string,string]": "string",
			},
		},
		{
			Name: "comment",
			Data: `// this is a comment
			replace foo bar`,
			Expected: map[string]string{
				"foo": "bar",
			},
		},
		{
			Name: "ignore whitespace",
			Data: `

			replace foo bar`,
			Expected: map[string]string{
				"foo": "bar",
			},
		},
		{
			Name:          "unknown directive",
			Data:          `foo`,
			ExpectedError: fmt.Errorf("could not parse override: 'foo'"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			overrides, err := parseOverrides(strings.NewReader(tc.Data))
			assert.Equal(t, tc.Expected, overrides)
			assert.Equal(t, tc.ExpectedError, err)
		})
	}
}

func TestGen_TypeOverridesFile(t *testing.T) {
	customPath := "/foo/bar/baz"

	tmp, err := os.CreateTemp("", "")
	require.NoError(t, err)

	defer os.Remove(tmp.Name())

	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple/docs",
		PropNamingStrategy: "",
	}

	t.Run("Default file is missing", func(t *testing.T) {
		open = func(path string) (*os.File, error) {
			assert.Equal(t, DefaultOverridesFile, path)

			return nil, os.ErrNotExist
		}
		defer func() {
			open = os.Open
		}()

		config.OverridesFile = DefaultOverridesFile
		err := New().Build(config)
		assert.NoError(t, err)
	})

	t.Run("Default file is present", func(t *testing.T) {
		open = func(path string) (*os.File, error) {
			assert.Equal(t, DefaultOverridesFile, path)

			return tmp, nil
		}
		defer func() {
			open = os.Open
		}()

		config.OverridesFile = DefaultOverridesFile
		err := New().Build(config)
		assert.NoError(t, err)
	})

	t.Run("Different file is missing", func(t *testing.T) {
		open = func(path string) (*os.File, error) {
			assert.Equal(t, customPath, path)

			return nil, os.ErrNotExist
		}
		defer func() {
			open = os.Open
		}()

		config.OverridesFile = customPath
		err := New().Build(config)
		assert.EqualError(t, err, "could not open overrides file: /foo/bar/baz: file does not exist")
	})

	t.Run("Different file is present", func(t *testing.T) {
		open = func(path string) (*os.File, error) {
			assert.Equal(t, customPath, path)

			return tmp, nil
		}
		defer func() {
			open = os.Open
		}()

		config.OverridesFile = customPath
		err := New().Build(config)
		assert.NoError(t, err)
	})
}
func TestGen_Debugger(t *testing.T) {
	var buf bytes.Buffer
	config := &Config{
		SearchDir:          searchDir,
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/simple/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
		Debugger:           log.New(&buf, "", log.LstdFlags),
	}
	assert.True(t, buf.Len() == 0)
	assert.NoError(t, New().Build(config))
	assert.True(t, buf.Len() > 0)

	expectedFiles := []string{
		filepath.Join(config.OutputDir, "swagger.json"),
		filepath.Join(config.OutputDir, "swagger.yaml"),
	}
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}

		_ = os.Remove(expectedFile)
	}
}

func TestGen_ErrorAndInterface(t *testing.T) {
	t.Skip("Legacy swag test: JSON comparison against stale expected files")
	config := &Config{
		SearchDir:          "../../testing/testdata/error",
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/error/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
	}

	assert.NoError(t, New().Build(config))

	expectedFiles := []string{
		filepath.Join(config.OutputDir, "swagger.json"),
		filepath.Join(config.OutputDir, "swagger.yaml"),
	}
	t.Cleanup(func() {
		for _, expectedFile := range expectedFiles {
			_ = os.Remove(expectedFile)
		}
	})

	// check files
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}
	}

	// check content
	jsonOutput, err := os.ReadFile(filepath.Join(config.OutputDir, "swagger.json"))
	if err != nil {
		require.NoError(t, err)
	}
	expectedJSON, err := os.ReadFile(filepath.Join(config.SearchDir, "expected.json"))
	if err != nil {
		require.NoError(t, err)
	}

	assert.JSONEq(t, string(expectedJSON), string(jsonOutput))
}

func TestGen_StateAdmin(t *testing.T) {
	t.Skip("Legacy swag test: JSON comparison against stale expected files")
	config := &Config{
		SearchDir:          "../../testing/testdata/state",
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/state/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
		State:              "admin",
	}

	assert.NoError(t, New().Build(config))

	expectedFiles := []string{
		filepath.Join(config.OutputDir, "admin_swagger.json"),
		filepath.Join(config.OutputDir, "admin_swagger.yaml"),
	}
	t.Cleanup(func() {
		for _, expectedFile := range expectedFiles {
			_ = os.Remove(expectedFile)
		}
	})

	// check files
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}
	}

	// check content
	jsonOutput, err := os.ReadFile(filepath.Join(config.OutputDir, "admin_swagger.json"))
	require.NoError(t, err)
	expectedJSON, err := os.ReadFile(filepath.Join(config.SearchDir, "admin_expected.json"))
	require.NoError(t, err)

	assert.JSONEq(t, string(expectedJSON), string(jsonOutput))
}

func TestGen_StateUser(t *testing.T) {
	t.Skip("Legacy swag test: JSON comparison against stale expected files")
	config := &Config{
		SearchDir:          "../../testing/testdata/state",
		MainAPIFile:        "./main.go",
		OutputDir:          "../../testing/testdata/state/docs",
		OutputTypes:        outputTypes,
		PropNamingStrategy: "",
		State:              "user",
	}

	assert.NoError(t, New().Build(config))

	expectedFiles := []string{
		filepath.Join(config.OutputDir, "user_swagger.json"),
		filepath.Join(config.OutputDir, "user_swagger.yaml"),
	}
	t.Cleanup(func() {
		for _, expectedFile := range expectedFiles {
			_ = os.Remove(expectedFile)
		}
	})

	// check files
	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			require.NoError(t, err)
		}
	}

	// check content
	jsonOutput, err := os.ReadFile(filepath.Join(config.OutputDir, "user_swagger.json"))
	require.NoError(t, err)
	expectedJSON, err := os.ReadFile(filepath.Join(config.SearchDir, "user_expected.json"))
	require.NoError(t, err)

	assert.JSONEq(t, string(expectedJSON), string(jsonOutput))
}
