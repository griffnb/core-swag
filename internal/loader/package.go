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

	// Last resort: derive import path from the module name via `go list -m`.
	// Directories without .go files (e.g., "applications/") fail both go list
	// and build.ImportDir. We get the module path from the Go toolchain and
	// combine it with the relative path from the module root to searchDir.
	abs, err := filepath.Abs(searchDir)
	if err == nil {
		if modPath, modDir := getModuleInfo(abs); modPath != "" {
			rel, relErr := filepath.Rel(modDir, abs)
			if relErr == nil && rel != "." {
				return modPath + "/" + filepath.ToSlash(rel), nil
			}
			if relErr == nil {
				return modPath, nil
			}
		}
	}

	return "", fmt.Errorf("failed to get package name for directory: %s", searchDir)
}

// getModuleInfo uses `go list -m` to get the module path and `go env GOMOD`
// to get the module root directory. Returns empty strings if either fails.
func getModuleInfo(dir string) (modulePath string, moduleDir string) {
	// Get module path
	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", ""
	}
	modPath := strings.TrimSpace(string(out))
	if modPath == "" {
		return "", ""
	}

	// Get module root directory via go env GOMOD (path to go.mod)
	cmd = exec.Command("go", "env", "GOMOD")
	cmd.Dir = dir
	out, err = cmd.Output()
	if err != nil {
		return "", ""
	}
	gomodFile := strings.TrimSpace(string(out))
	if gomodFile == "" {
		return "", ""
	}

	return modPath, filepath.Dir(gomodFile)
}
