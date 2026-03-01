// Package orchestrator coordinates all services to generate OpenAPI documentation.
package orchestrator

import (
	"fmt"
	"go/ast"
	"runtime"
	"sort"
	"sync"

	"github.com/go-openapi/spec"
	"github.com/griffnb/core-swag/internal/loader"
	"github.com/griffnb/core-swag/internal/parser/route"
	routedomain "github.com/griffnb/core-swag/internal/parser/route/domain"
	"golang.org/x/sync/errgroup"
)

// fileRoutes pairs a file path with its parsed routes for deterministic ordering.
type fileRoutes struct {
	filePath string
	routes   []*routedomain.Route
}

// parseRoutesParallel parses routes from all files concurrently using an errgroup
// bounded by the number of CPUs. Results are sorted by file path to ensure
// deterministic output regardless of goroutine scheduling order.
// Returns the accumulated routes, the total operation count, and any error.
func (s *Service) parseRoutesParallel(files map[*ast.File]*loader.AstFileInfo) ([]*routedomain.Route, int, error) {
	var (
		mu       sync.Mutex
		collected []fileRoutes
	)

	var g errgroup.Group
	g.SetLimit(runtime.NumCPU())

	for astFile, fileInfo := range files {
		if astFile == nil {
			continue
		}

		// Capture loop variables for the goroutine closure.
		astFile, fileInfo := astFile, fileInfo

		g.Go(func() error {
			routes, err := s.routeParser.ParseRoutes(astFile, fileInfo.Path, fileInfo.FileSet)
			if err != nil {
				return fmt.Errorf("failed to parse routes from %s: %w", fileInfo.Path, err)
			}
			if len(routes) == 0 {
				return nil
			}

			mu.Lock()
			collected = append(collected, fileRoutes{
				filePath: fileInfo.Path,
				routes:   routes,
			})
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, 0, err
	}

	// Sort by file path for deterministic output.
	sort.Slice(collected, func(i, j int) bool {
		return collected[i].filePath < collected[j].filePath
	})

	// Merge sorted results into swagger.Paths sequentially.
	var allRoutes []*routedomain.Route
	routeCount := 0

	for _, fr := range collected {
		allRoutes = append(allRoutes, fr.routes...)

		for _, r := range fr.routes {
			operation := route.RouteToSpecOperation(r)
			if operation == nil {
				continue
			}

			s.ensureSwaggerPaths()

			pathItem := s.swagger.Paths.Paths[r.Path]

			switch r.Method {
			case "GET":
				pathItem.Get = operation
			case "POST":
				pathItem.Post = operation
			case "PUT":
				pathItem.Put = operation
			case "DELETE":
				pathItem.Delete = operation
			case "PATCH":
				pathItem.Patch = operation
			case "OPTIONS":
				pathItem.Options = operation
			case "HEAD":
				pathItem.Head = operation
			}

			s.swagger.Paths.Paths[r.Path] = pathItem
			routeCount++
		}
	}

	return allRoutes, routeCount, nil
}

// ensureSwaggerPaths initializes the swagger Paths map if it has not been created yet.
func (s *Service) ensureSwaggerPaths() {
	if s.swagger.Paths == nil {
		s.swagger.Paths = &spec.Paths{Paths: make(map[string]spec.PathItem)}
	}
	if s.swagger.Paths.Paths == nil {
		s.swagger.Paths.Paths = make(map[string]spec.PathItem)
	}
}
