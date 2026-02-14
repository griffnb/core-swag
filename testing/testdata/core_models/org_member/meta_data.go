package org_member

import "github.com/griffnb/core/lib/types"

type MetaData struct {
	PendingRemoval int64      `public:"view" json:"pending_removal"`
	PendingLocalID types.UUID `public:"view" json:"pending_local_id"`
	UnknownLocal   int64      `public:"view" json:"unknown_local"`
	PendingPlanID  types.UUID `public:"view" json:"pending_plan_id"`
}
