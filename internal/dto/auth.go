package dto

// LoginInput represents the login operation request.
type LoginInput struct {
	Body struct {
		Username string `json:"username" example:"alice" doc:"Username"`
		Password string `json:"password" example:"hunter2" doc:"Password"`
	}
}

// TokenPair is the login/refresh operation response data.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// RefreshInput represents the refresh-token operation request.
type RefreshInput struct {
	Body struct {
		RefreshToken string `json:"refreshToken"`
	}
}

// LogoutInput represents the logout operation request. RefreshToken is
// optional — if sent, it's revoked too, otherwise only the access token is.
type LogoutInput struct {
	Authorization string `header:"Authorization" doc:"Bearer access token"`
	Body          struct {
		RefreshToken string `json:"refreshToken,omitempty" doc:"Optional, also revokes the refresh token"`
	}
}

// CheckAuthInput represents the check-auth operation request.
type CheckAuthInput struct {
	Authorization string `header:"Authorization" doc:"Bearer access token"`
}

// AuthIdentity is the check-auth operation response data.
type AuthIdentity struct {
	UserID string   `json:"userId"`
	Roles  []string `json:"roles"`
}
