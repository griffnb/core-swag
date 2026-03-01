package field

// TransToValidCollectionFormat validates and returns a collection format string.
// Returns empty string if the format is not valid.
func TransToValidCollectionFormat(format string) string {
	switch format {
	case "csv", "multi", "pipes", "tsv", "ssv":
		return format
	}

	return ""
}
