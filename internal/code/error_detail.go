// Package code holds the stable, machine-readable codes passed to
// serviceerr.Error.AddDetail — clients match on these, not on Message.
package code

const (
	UserUsernameExisted = "USER_USERNAME_EXISTED"
	UserEmailExisted    = "USER_EMAIL_EXISTED"
	GroupNameExisted    = "GROUP_NAME_EXISTED"
)
