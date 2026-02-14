package field

import "unicode"

// ToSnakeCase converts a name to snake_case
func ToSnakeCase(in string) string {
	var (
		runes  = []rune(in)
		length = len(runes)
		out    []rune
	)

	for idx := 0; idx < length; idx++ {
		if idx > 0 && unicode.IsUpper(runes[idx]) &&
			((idx+1 < length && unicode.IsLower(runes[idx+1])) || unicode.IsLower(runes[idx-1])) {
			out = append(out, '_')
		}

		out = append(out, unicode.ToLower(runes[idx]))
	}

	return string(out)
}

// ToLowerCamelCase converts a name to lowerCamelCase
func ToLowerCamelCase(in string) string {
	var flag bool

	out := make([]rune, len(in))

	runes := []rune(in)
	for i, curr := range runes {
		if (i == 0 && unicode.IsUpper(curr)) || (flag && unicode.IsUpper(curr)) {
			out[i] = unicode.ToLower(curr)
			flag = true

			continue
		}

		out[i] = curr
		flag = false
	}

	return string(out)
}

// ApplyNamingStrategy applies the specified naming strategy to a field name
func ApplyNamingStrategy(name string, strategy string) string {
	switch strategy {
	case SnakeCase:
		return ToSnakeCase(name)
	case PascalCase:
		return name
	default:
		return ToLowerCamelCase(name)
	}
}
