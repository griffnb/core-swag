package embedded_tag_test

import "github.com/griffnb/core/lib/model/fields"

// BaseModelLike mimics fields that have no tags (like BaseModel private fields)
type BaseModelLike struct {
	ChangeLogs  bool   // No tags - should be excluded
	Client      string // No tags - should be excluded
	ManualCache string // No tags - should be excluded
}

// TestModel demonstrates field filtering
type TestModel struct {
	BaseModelLike
	ValidJSON       *fields.StringField `json:"valid_json"`             // Has json tag - should be included
	ValidColumn     *fields.StringField `column:"valid_column"`         // Has column tag - should be included
	BothTags        *fields.StringField `json:"both" column:"both_col"` // Has both tags - should be included
	ExcludedJSON    *fields.StringField `json:"-"`                      // Explicitly excluded with json:"-"
	ExcludedColumn  *fields.StringField `column:"-"`                    // Explicitly excluded with column:"-"
	NoTags          string              // No tags - should be excluded
	PublicWithJSON  *fields.StringField `json:"public_field" public:"view"` // Has json and public - should be included
	PublicNoJSON    *fields.StringField `public:"view"`                     // Only public tag, no json/column - should be excluded
}
