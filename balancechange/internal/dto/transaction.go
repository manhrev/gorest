package dto

import "time"

// Transaction is the canonical transaction representation returned by the
// API. JSON keys are snake_case (not huma-bob's camelCase) to match the
// request contract the bank-notification pusher already speaks.
type Transaction struct {
	ID         string    `json:"id" format:"uuid" doc:"Transaction ID (UUID)"`
	Note       string    `json:"note"`
	Amount     int64     `json:"amount" doc:"VND, always positive; direction is in type"`
	Type       string    `json:"type" enum:"income,expense"`
	Category   string    `json:"category" enum:"food_drink,entertainment,shopping,other"`
	Owner      string    `json:"owner" enum:"van,manh"`
	RawText    string    `json:"raw_text"`
	CreatedAt  time.Time `json:"created_at"`
	InsertedAt time.Time `json:"inserted_at"`
}

// CreateTransactionInput represents the create-transaction operation request.
type CreateTransactionInput struct {
	Body struct {
		RawText   string    `json:"raw_text" minLength:"1" maxLength:"4000" doc:"Raw bank notification text"`
		CreatedAt time.Time `json:"created_at" doc:"When the balance change happened (RFC 3339)"`
		Owner     string    `json:"owner,omitempty" enum:"van,manh" doc:"Overrides Gemini's guess, omit to let Gemini infer"`
	}
}
