// Package code holds the stable, machine-readable codes passed to
// serviceerr.Error.AddDetail — clients match on these, not on Message.
package code

const (
	TransactionDuplicated   = "TRANSACTION_DUPLICATED"
	TransactionNotParseable = "TRANSACTION_NOT_PARSEABLE"
)
