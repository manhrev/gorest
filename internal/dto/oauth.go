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
	Authorization string `header:"Authorization" doc:"Bearer access token of the already-logged-in user"`
	ClientID      string `query:"client_id"`
	RedirectURI   string `query:"redirect_uri"`
	ResponseType  string `query:"response_type" doc:"Must be \"code\""`
	Scope         string `query:"scope,omitempty"`
	State         string `query:"state,omitempty"`
}

// OAuthAuthorizeOutput is a redirect (302) to redirect_uri?code=...&state=...
type OAuthAuthorizeOutput struct {
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
