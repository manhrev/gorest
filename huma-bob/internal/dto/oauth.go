package dto

// OAuthAuthorizeInput represents the OAuth2 authorize operation request
// (RFC 6749 §4.1.1).
//
// Caveat: the user is identified via the Authorization bearer header, same
// as check-auth — NOT a session cookie. A real browser top-level redirect
// can't carry a custom header, so this only works when whatever calls this
// endpoint already holds the user's access token itself (a script/fetch
// call), not an <a href>/window.location navigation. See pkg/oauthserver's
// package doc for the full tradeoff.
type OAuthAuthorizeInput struct {
	Authorization       string `header:"Authorization" doc:"Bearer access token of the already-logged-in user"`
	ClientID            string `query:"client_id"`
	RedirectURI         string `query:"redirect_uri"`
	ResponseType        string `query:"response_type" doc:"Must be \"code\""`
	Scope               string `query:"scope,omitempty"`
	State               string `query:"state,omitempty"`
	CodeChallenge       string `query:"code_challenge" doc:"RFC 7636 PKCE challenge, required"`
	CodeChallengeMethod string `query:"code_challenge_method" doc:"Must be \"S256\""`
}

// OAuthAuthorizeOutput is either a redirect (302, Location set, Body zero
// value — ignore it) to redirect_uri?code=...&state=..., or, for a
// RequireConsent client, a 200 describing the pending request (Location
// empty) — the caller then collects an approve/deny decision and calls
// POST .../decision with Body.ConsentID.
type OAuthAuthorizeOutput struct {
	Status   int    `json:"-"`
	Location string `header:"Location"` // huma omits an empty header value automatically, no ",omitempty" modifier support on this tag
	Body     struct {
		ConsentRequired bool   `json:"consentRequired"`
		ConsentID       string `json:"consentId,omitempty"`
		ClientID        string `json:"clientId,omitempty"`
		Scope           string `json:"scope,omitempty"`
	}
}

// OAuthDecisionInput represents the consent decision operation request —
// not part of RFC 6749 itself (consent screens are left to implementations),
// this is this server's own API for collecting one.
type OAuthDecisionInput struct {
	Authorization string `header:"Authorization" doc:"Bearer access token of the user deciding"`
	Body          struct {
		ConsentID string `json:"consentId"`
		Approve   bool   `json:"approve"`
	}
}

// OAuthDecisionOutput is always a redirect: to redirect_uri?code=...&state=...
// on approve, or redirect_uri?error=access_denied&state=... on deny (RFC
// 6749 §4.1.2.1).
type OAuthDecisionOutput struct {
	Status   int    `json:"-"`
	Location string `header:"Location"`
}

// OAuthTokenInput represents the OAuth2 token operation request
// (RFC 6749 §4.1.3).
type OAuthTokenInput struct {
	Body struct {
		GrantType    string `json:"grant_type" doc:"Must be \"authorization_code\""`
		Code         string `json:"code"`
		RedirectURI  string `json:"redirect_uri"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		CodeVerifier string `json:"code_verifier" doc:"RFC 7636 PKCE verifier, required"`
	}
}

// OAuthTokenOutput deliberately does NOT use the app's response.Output[T]
// envelope — RFC 6749 §5.1 mandates this exact flat shape so any generic
// OAuth2 client library can parse it. One intentional inconsistency with
// the rest of the API, for standards compliance.
type OAuthTokenOutput struct {
	Body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
}
