package loader

import (
	"fmt"
	"go/build"
	"os/exec"
	"path/filepath"
	"strings"
)

// getPkgName returns the package import path for a directory
func getPkgName(searchDir string) (string, error) {
	// First try using go list command (more reliable for getting import paths)
	cmd := exec.Command("go", "list", "-f={{.ImportPath}}")
	cmd.Dir = searchDir

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		outStr := strings.TrimSpace(stdout.String())

		// Handle old GOPATH format
		if len(outStr) > 0 && outStr[0] == '_' {
			outStr = strings.TrimPrefix(outStr, "_"+build.Default.GOPATH+"/src/")
		}

		// Split by newline and take first line
		lines := strings.Split(outStr, "\n")
		if len(lines) > 0 && lines[0] != "" {
			return lines[0], nil
		}
	}

	// Fall back to build.ImportDir if go list fails
	if abs, err := filepath.Abs(searchDir); err == nil {
		pkg, err := build.ImportDir(abs, build.ImportComment)
		if err == nil {
			return pkg.ImportPath, nil
		}
	}

	// Last resort: return error
	return "", fmt.Errorf("failed to get package name for directory: %s", searchDir)
}
