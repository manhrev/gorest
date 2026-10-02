// Package converter holds model <-> dto translation, kept out of the
// service layer so it's reusable and independently testable.
package converter

import (
	"github.com/manhrev/gorest/balancechange/internal/db/model"
	"github.com/manhrev/gorest/balancechange/internal/dto"
)

func TransactionToDto(m *model.Transaction) dto.Transaction {
	return dto.Transaction{
		ID:         m.ID,
		Note:       m.Note,
		Amount:     m.Amount,
		Type:       m.Type,
		Category:   m.Category,
		Owner:      m.Owner,
		RawText:    m.RawText,
		CreatedAt:  m.CreatedAt,
		InsertedAt: m.InsertedAt,
	}
}
