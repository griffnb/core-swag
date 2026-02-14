package account

import (
	"time"

	"github.com/griffnb/core-swag/testing/testdata/core_models/constants"
	"github.com/griffnb/core-swag/testing/testdata/core_models/org_member"
	"github.com/griffnb/core/lib/model/fields"
	"github.com/griffnb/core/lib/types"
)

type OrgAccount struct {
	ID                                     types.UUID            `public:"view" json:"id,omitempty"`
	AccountID                              *types.UUID           `public:"view" json:"account_id,omitempty"`
	OrgMemberID                            *types.UUID           `public:"view" json:"org_member_id,omitempty"`
	Status                                 constants.Status      `public:"view" json:"status,omitempty"`
	Name                                   string                `public:"view" json:"name,omitempty"`
	FirstName                              string                `public:"view" json:"first_name,omitempty"`
	MiddleName                             string                `public:"view" json:"middle_name,omitempty"`
	LastName                               string                `public:"view" json:"last_name,omitempty"`
	OrganizationID                         types.UUID            `public:"view" json:"organization_id,omitempty"`
	OrganizationName                       string                `public:"view" json:"organization_name,omitempty"`
	UnionID                                types.UUID            `public:"view" json:"union_id,omitempty"`
	UnionLocalID                           types.UUID            `public:"view" json:"union_local_id,omitempty"`
	UnionLocalName                         string                `public:"view" json:"union_local_name,omitempty"`
	UnionLocalNumber                       string                `public:"view" json:"union_local_number,omitempty"`
	OrganizationVerification               int64                 `public:"view" json:"organization_verification,omitempty"`
	OnboardActionFinishedTS                int64                 `public:"view" json:"onboard_action_finished_ts,omitempty"`
	OrganizationSubscriptionPlanID         types.UUID            `public:"view" json:"organization_subscription_plan_id,omitempty"`
	OrganizationSubscriptionPlanName       string                `public:"view" json:"organization_subscription_plan_name,omitempty"`
	OrganizationSubscriptionPlanProperties map[string]any        `public:"view" json:"organization_subscription_plan_properties,omitempty"`
	CreatedAt                              *time.Time            `public:"view" json:"created_at,omitempty"`
	OrgMember                              *JoinedOrgMember      `public:"view" json:"org_member,omitempty"`
	OrgMemberLocalMemberID                 string                `public:"view" json:"org_member_local_member_id,omitempty"`
	OrgMemberUnionStatus                   constants.UnionStatus `public:"view" json:"org_member_union_status,omitempty"`
	OrgMemberStatus                        constants.Status      `public:"view" json:"org_member_status,omitempty"`
	OrgMemberEmail                         string                `public:"view" json:"org_member_email,omitempty"`
}

type JoinedOrgMember struct {
	ID               types.UUID            `public:"view" json:"id"`
	UnionID          string                `public:"view" json:"union_id"`
	UnionLocalID     types.UUID            `public:"view" json:"union_local_id"`
	UnionLocalName   string                `public:"view" json:"union_local_name"`
	UnionLocalNumber string                `public:"view" json:"union_local_number"`
	LocalMemberID    types.UUID            `public:"view" json:"local_member_id"`
	UnionStatus      constants.UnionStatus `public:"view" json:"union_status"`
	Status           constants.Status      `public:"view" json:"status"`
	MetaData         *org_member.MetaData  `public:"view" json:"meta_data"`
}

type OrgManagementJoins struct {
	OrgMemberLocalMemberID *fields.StringField `json:"org_member_local_member_id" type:"text"`
	OrgMemberID            *fields.UUIDField   `json:"org_member_id"              type:"uuid"`

	// Org member joins, needed for filtering
	OrgMemberStatus      *fields.IntConstantField[constants.Status]      `json:"org_member_status"       type:"smallint"`
	OrgMemberUnionStatus *fields.IntConstantField[constants.UnionStatus] `json:"org_member_union_status" type:"smallint"`
	OrgMemberEmail       *fields.StringField                             `json:"org_member_email"        type:"text"`
}
