package account

import "github.com/griffnb/core/lib/types"

type Flags struct {
	ForcePendingRestartFlow int64      `public:"view" json:"force_pending_restart_flow,omitempty"` // force them into flows
	PendingRestartFlow      int64      `public:"view" json:"pending_restart_flow,omitempty"`       // Needs to do flows
	SendDSRs                int64      `public:"view" json:"send_dsrs,omitempty"`                  // Has DSRs that need to go out
	NeverSentDSRs           int64      `public:"view" json:"never_sent_dsrs,omitempty"`            // Has never sent DSRs
	NoCoverage              int64      `public:"view" json:"no_coverage"`                          // Lost coverage
	NewClassificationID     types.UUID `public:"view" json:"new_classification_id"`                // New Classification ID if changed
	TermsNeeded             int64      `public:"view" json:"terms_needed,omitempty"`               // terms to sign
	SimplifyLogin           int64      `public:"view" json:"simplify_login,omitempty"`             // show simplify login alert

	SharedAddressUpdates map[types.UUID]*SharedAddressUpdate `public:"view" json:"shared_address_update,omitempty"`
	AddressUpdateBy      string                              `public:"view" json:"address_update_by,omitempty"`

	AuthorizationNeeded int64 `public:"view" json:"authorization_needed,omitempty"` // used by orgs for account validation

	ShowNewDataBrokers int64 `public:"view" json:"show_new_data_brokers,omitempty"` // shows alert for brokers
	NewDataBrokers     int64 `public:"view" json:"new_data_brokers,omitempty"`      // number of new brokers
	LastBrokerEmailTS  int64 `              json:"last_broker_email_ts,omitempty"`  // used for last time they got a broker email, not used currently

	LexisFreeze int64 `public:"view" json:"lexis_freeze,omitempty"` // Special Case

	// not currently used
	ClassificationChanged int64 `public:"view" json:"classification_changed,omitempty"`
	WelcomeBack           int64 `public:"view" json:"welcome_back,omitempty"`
}

type SharedAddressUpdate struct {
	UpdateType SharedAddressUpdateType `public:"view" json:"update_type"`
	ID         types.UUID              `public:"view" json:"id"`
	PreviousID types.UUID              `public:"view" json:"previous_id"`
}

type SharedAddressUpdateType int

const (
	UPDATE_TYPE_NEW SharedAddressUpdateType = iota + 1
	UPDATE_TYPE_UPDATE
	UPDATE_TYPE_DELETE
)
