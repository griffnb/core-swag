package data

import (
	"time"

	"github.com/griffnb/core-swag/testing/testdata/alias_type/types"
)

type TimeContainer struct {
	Name      types.StringAlias `json:"name"`
	Timestamp time.Time         `json:"timestamp"`
	CreatedAt types.DateOnly    `json:"created_at"`
}
