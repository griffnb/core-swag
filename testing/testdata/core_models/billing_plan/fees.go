package billing_plan

import "github.com/shopspring/decimal"

type Fees struct {
	SetupRate   decimal.Decimal `json:"setup_rate"`
	MonthlyRate decimal.Decimal `json:"monthly_rate"`
	SMSRate     decimal.Decimal `json:"sms_rate"`
	MMSRate     decimal.Decimal `json:"mms_rate"`
	VoiceRate   decimal.Decimal `json:"voice_rate"`
}
