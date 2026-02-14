package field

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// structField holds parsed field information from struct tags.
type structField struct {
	title        string
	schemaType   string
	arrayType    string
	formatType   string
	maximum      *float64
	minimum      *float64
	multipleOf   *float64
	maxLength    *int64
	minLength    *int64
	maxItems     *int64
	minItems     *int64
	exampleValue interface{}
	enums        []interface{}
	enumVarNames []interface{}
	unique       bool
}

// getFloatTag extracts a float value from a struct tag.
func getFloatTag(structTag reflect.StructTag, tagName string) (*float64, error) {
	strValue := structTag.Get(tagName)
	if strValue == "" {
		return nil, nil
	}

	value, err := strconv.ParseFloat(strValue, 64)
	if err != nil {
		return nil, fmt.Errorf("can't parse numeric value of %q tag: %v", tagName, err)
	}

	return &value, nil
}

// getIntTag extracts an int value from a struct tag.
func getIntTag(structTag reflect.StructTag, tagName string) (*int64, error) {
	strValue := structTag.Get(tagName)
	if strValue == "" {
		return nil, nil
	}

	value, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("can't parse numeric value of %q tag: %v", tagName, err)
	}

	return &value, nil
}

// parseValidTags parses validation tags (binding, validate) and populates the structField.
func parseValidTags(validTag string, sf *structField) {
	// `validate:"required,max=10,min=1"`
	// ps. required checked by IsRequired().
	for _, val := range strings.Split(validTag, ",") {
		var (
			valValue string
			keyVal   = strings.Split(val, "=")
		)

		switch len(keyVal) {
		case 1:
		case 2:
			valValue = strings.ReplaceAll(strings.ReplaceAll(keyVal[1], utf8HexComma, ","), utf8Pipe, "|")
		default:
			continue
		}

		switch keyVal[0] {
		case "max", "lte":
			sf.setMax(valValue)
		case "min", "gte":
			sf.setMin(valValue)
		case "oneof":
			sf.setOneOf(valValue)
		case "unique":
			if sf.schemaType == ARRAY {
				sf.unique = true
			}
		case "dive":
			// ignore dive
			return
		default:
			continue
		}
	}
}

// setMin sets the minimum value constraint.
func (sf *structField) setMin(valValue string) {
	value, err := strconv.ParseFloat(valValue, 64)
	if err != nil {
		return
	}

	switch sf.schemaType {
	case INTEGER, NUMBER:
		sf.minimum = &value
	case STRING:
		intValue := int64(value)
		sf.minLength = &intValue
	case ARRAY:
		intValue := int64(value)
		sf.minItems = &intValue
	}
}

// setMax sets the maximum value constraint.
func (sf *structField) setMax(valValue string) {
	value, err := strconv.ParseFloat(valValue, 64)
	if err != nil {
		return
	}

	switch sf.schemaType {
	case INTEGER, NUMBER:
		sf.maximum = &value
	case STRING:
		intValue := int64(value)
		sf.maxLength = &intValue
	case ARRAY:
		intValue := int64(value)
		sf.maxItems = &intValue
	}
}

// setOneOf sets enum values from oneof validation tag.
// NOTE: This sets string values - the actual type conversion happens later
// in the field parser's parseEnumTags method which has access to SchemaHelper.
func (sf *structField) setOneOf(valValue string) {
	if len(sf.enums) != 0 {
		return
	}

	valValues := parseOneOfParam(valValue)
	for i := range valValues {
		sf.enums = append(sf.enums, valValues[i])
	}
}

// Cache for oneOf parameter parsing
var (
	oneofValsCache       = map[string][]string{}
	oneofValsCacheRWLock = sync.RWMutex{}
	splitParamsRegex     = regexp.MustCompile(`'[^']*'|\S+`)
)

// parseOneOfParam parses the oneof validation parameter.
// Code copied from github.com/go-playground/validator
func parseOneOfParam(param string) []string {
	oneofValsCacheRWLock.RLock()
	values, ok := oneofValsCache[param]
	oneofValsCacheRWLock.RUnlock()

	if !ok {
		oneofValsCacheRWLock.Lock()
		values = splitParamsRegex.FindAllString(param, -1)

		for i := 0; i < len(values); i++ {
			values[i] = strings.ReplaceAll(values[i], "'", "")
		}

		oneofValsCache[param] = values

		oneofValsCacheRWLock.Unlock()
	}

	return values
}

// splitNotWrapped slices s into all substrings separated by sep if sep is not
// wrapped by brackets and returns a slice of the substrings between those separators.
func splitNotWrapped(s string, sep rune) []string {
	openCloseMap := map[rune]rune{
		'(': ')',
		'[': ']',
		'{': '}',
	}

	var (
		result    = make([]string, 0)
		current   = strings.Builder{}
		openCount = 0
		openChar  rune
	)

	for _, char := range s {
		switch {
		case openChar == 0 && openCloseMap[char] != 0:
			openChar = char

			openCount++

			current.WriteRune(char)
		case char == openChar:
			openCount++

			current.WriteRune(char)
		case openCount > 0 && char == openCloseMap[openChar]:
			openCount--

			current.WriteRune(char)
		case openCount == 0 && char == sep:
			result = append(result, current.String())

			openChar = 0

			current = strings.Builder{}
		default:
			current.WriteRune(char)
		}
	}

	if current.String() != "" {
		result = append(result, current.String())
	}

	return result
}
