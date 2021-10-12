package auth

// GetToken uses the provided authenticator to get a token using the
// appropriate flow depending on the provided options
func GetToken(options *Options, authenticator Authenticator) (*Token, error) {
	if options.User != "" && options.Password != "" {
		return authenticator.PasswordCredentials(options.User, options.Password)
	}
	return authenticator.ClientCredentials()
}
