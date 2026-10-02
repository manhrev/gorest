// Package gemini turns a raw bank balance-change notification into
// structured fields via the Gemini REST API (generateContent with a JSON
// Schema response). Output is still untrusted: the service layer
// re-validates every field before it reaches the DB.
package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/manhrev/gorest/balancechange/internal/enum"
	"github.com/manhrev/mkit/httpclient"
)

const baseURL = "https://generativelanguage.googleapis.com/v1beta/models/"

const systemPrompt = `You parse Vietnamese bank balance-change notifications (SMS / app push) into JSON.

Rules:
- is_transaction: false if the text is not a balance change (OTP, promotion, login alert, ...). Fill the other fields with any valid value in that case.
- amount: the changed amount in VND as a positive integer. Vietnamese banks use "." or "," as thousands separators ("1.500.000" or "1,500,000" = 1500000). Never the remaining balance ("SD", "So du").
- type: "income" for money in ("+", "GD: +", "nhan", "credit"), "expense" for money out ("-", "thanh toan", "chuyen di", "debit").
- category: "food_drink" (an uong: restaurants, cafes, food delivery, groceries for meals), "entertainment" (giai tri: movies, games, streaming, travel), "shopping" (mua sam: retail, e-commerce like Shopee/Lazada/Tiki), otherwise "other" (transfers, bills, salary, ...).
- owner: "van" if the text mentions Van/Vân as the account holder, otherwise "manh".
- note: short English summary of the transaction description (merchant / transfer message), max 100 chars.`

// Parsed is Gemini's structured reading of one notification.
type Parsed struct {
	IsTransaction bool   `json:"is_transaction"`
	Amount        int64  `json:"amount"`
	Type          string `json:"type"`
	Category      string `json:"category"`
	Owner         string `json:"owner"`
	Note          string `json:"note"`
}

type Client struct {
	http   *httpclient.Client
	apiKey string
	model  string
}

func New(apiKey, model string) *Client {
	return &Client{
		http:   httpclient.NewClient(&httpclient.Config{DefaultTimeout: 30 * time.Second, DefaultRetryCount: 2}),
		apiKey: apiKey,
		model:  model,
	}
}

type generateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text    string `json:"text"`
				Thought bool   `json:"thought"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

func (c *Client) Parse(ctx context.Context, rawText string) (Parsed, error) {
	body := map[string]any{
		"systemInstruction": map[string]any{"parts": []any{map[string]any{"text": systemPrompt}}},
		"contents":          []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": rawText}}}},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			// responseJsonSchema = standard JSON Schema (lowercase types);
			// responseSchema would need Gemini's uppercase OBJECT/STRING enum.
			"responseJsonSchema": responseSchema,
		},
	}

	var out generateResponse
	// key in header, not ?key=, so it never lands in URL/access logs.
	res, err := c.http.Req().
		SetHeader("x-goog-api-key", c.apiKey).
		SetBody(body).
		SetResult(&out).
		Post(ctx, baseURL+c.model+":generateContent")
	if err != nil {
		return Parsed{}, fmt.Errorf("gemini request: %w", err)
	}
	defer res.RawResponse.Body.Close()
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.RawResponse.Body, 1024))
		return Parsed{}, fmt.Errorf("gemini status %d: %s", res.StatusCode, msg)
	}

	if len(out.Candidates) == 0 {
		return Parsed{}, errors.New("gemini returned no candidates")
	}
	var text strings.Builder
	for _, p := range out.Candidates[0].Content.Parts {
		if !p.Thought {
			text.WriteString(p.Text)
		}
	}

	var p Parsed
	if err := json.Unmarshal([]byte(text.String()), &p); err != nil {
		return Parsed{}, fmt.Errorf("gemini output not JSON (finishReason=%s): %w", out.Candidates[0].FinishReason, err)
	}

	return p, nil
}

// responseSchema enums come from internal/enum, the same lists the
// service validates against and the DB CHECKs mirror.
var responseSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"is_transaction": map[string]any{"type": "boolean"},
		"amount":         map[string]any{"type": "integer"},
		"type":           map[string]any{"type": "string", "enum": enum.Types},
		"category":       map[string]any{"type": "string", "enum": enum.Categories},
		"owner":          map[string]any{"type": "string", "enum": enum.Owners},
		"note":           map[string]any{"type": "string"},
	},
	"required": []string{"is_transaction", "amount", "type", "category", "owner", "note"},
}
