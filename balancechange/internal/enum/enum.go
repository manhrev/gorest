// Package enum holds the allowed values for transactions' TEXT enum
// columns. The DB enforces the same lists via CHECK constraints (see
// migrations); change both together.
package enum

const (
	TypeIncome  = "income"
	TypeExpense = "expense"

	CategoryFoodDrink     = "food_drink"
	CategoryEntertainment = "entertainment"
	CategoryShopping      = "shopping"
	CategoryOther         = "other"

	OwnerVan  = "van"
	OwnerManh = "manh"
)

var (
	Types      = []string{TypeIncome, TypeExpense}
	Categories = []string{CategoryFoodDrink, CategoryEntertainment, CategoryShopping, CategoryOther}
	Owners     = []string{OwnerVan, OwnerManh}
)
