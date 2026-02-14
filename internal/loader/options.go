package loader

// NewService creates a new loader service with optional configuration
func NewService(options ...Option) *Service {
	s := &Service{
		parseVendor:     false,
		parseInternal:   false,
		excludes:        make(map[string]struct{}),
		packagePrefix:   []string{},
		parseExtension:  ".go",
		useGoList:       false,
		useGoPackages:   false,
		parseDependency: ParseNone,
		debug:           &noOpDebugger{},
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

// WithParseVendor sets whether to parse vendor directories
func WithParseVendor(parse bool) Option {
	return func(s *Service) {
		s.parseVendor = parse
	}
}

// WithParseInternal sets whether to parse internal packages
func WithParseInternal(parse bool) Option {
	return func(s *Service) {
		s.parseInternal = parse
	}
}

// WithExcludes sets directory exclusion patterns
func WithExcludes(excludes map[string]struct{}) Option {
	return func(s *Service) {
		s.excludes = excludes
	}
}

// WithPackagePrefix sets package path prefixes to filter
func WithPackagePrefix(prefixes []string) Option {
	return func(s *Service) {
		s.packagePrefix = prefixes
	}
}

// WithParseExtension sets the file extension to parse
func WithParseExtension(ext string) Option {
	return func(s *Service) {
		s.parseExtension = ext
	}
}

// WithGoList sets whether to use go list for dependencies
func WithGoList(use bool) Option {
	return func(s *Service) {
		s.useGoList = use
	}
}

// WithGoPackages sets whether to use go/packages
func WithGoPackages(use bool) Option {
	return func(s *Service) {
		s.useGoPackages = use
	}
}

// WithParseDependency sets the dependency parsing flag
func WithParseDependency(flag ParseFlag) Option {
	return func(s *Service) {
		s.parseDependency = flag
	}
}

// WithDebugger sets the debugger for logging
func WithDebugger(debugger Debugger) Option {
	return func(s *Service) {
		s.debug = debugger
	}
}
