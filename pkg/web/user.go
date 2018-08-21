package web

// TokenData represents the authentication token
//go:generate counterfeiter . TokenData
type TokenData interface {
	// Claims reads the claims from the token into the specified struct
	Claims(v interface{}) error
}

// User holds the information for the current user
type User struct {
	Name string `json:"name"`
	TokenData
}
