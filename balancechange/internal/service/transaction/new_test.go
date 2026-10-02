package transaction

import (
	"testing"
	"time"

	"github.com/manhrev/gorest/balancechange/internal/gemini"
)

func TestBuildSetter(t *testing.T) {
	ok := gemini.Parsed{IsTransaction: true, Amount: 50000, Type: "expense", Category: "food_drink", Owner: "manh", Note: "Highlands"}
	now := time.Now()

	s, err := buildSetter(ok, "raw", now, "")
	if err != nil || *s.Amount != 50000 || *s.Owner != "manh" || *s.Category != "food_drink" {
		t.Fatalf("valid parse: %v %+v", err, s)
	}

	if s, _ := buildSetter(ok, "raw", now, "van"); *s.Owner != "van" {
		t.Fatalf("owner override ignored: %s", *s.Owner)
	}

	bad := map[string]func(*gemini.Parsed){
		"not transaction": func(p *gemini.Parsed) { p.IsTransaction = false },
		"zero amount":     func(p *gemini.Parsed) { p.Amount = 0 },
		"negative amount": func(p *gemini.Parsed) { p.Amount = -1 },
		"bad type":        func(p *gemini.Parsed) { p.Type = "spent" },
		"bad category":    func(p *gemini.Parsed) { p.Category = "an_uong" },
		"bad owner":       func(p *gemini.Parsed) { p.Owner = "bob" },
	}
	for name, mutate := range bad {
		p := ok
		mutate(&p)
		if _, err := buildSetter(p, "raw", now, ""); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}
