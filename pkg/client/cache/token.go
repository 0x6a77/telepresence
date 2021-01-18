package cache

import (
	"golang.org/x/oauth2"
)

const (
	tokenFile = "tokens.json"
)

// SaveTokenToUserCache saves the provided token to user cache and returns an error if something
// goes wrong while marshalling or persisting.
func SaveTokenToUserCache(token *oauth2.Token) error {
	return saveToUserCache(token, tokenFile)
}

// LoadTokenFromUserCache gets the token instance from cache or returns an error if something goes
// wrong while loading or unmarshalling.
func LoadTokenFromUserCache() (*oauth2.Token, error) {
	var token oauth2.Token
	err := loadFromUserCache(&token, tokenFile)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteTokenFromUserCache removes token cache if existing or returns an error
func DeleteTokenFromUserCache() error {
	return deleteFromUserCache(tokenFile)
}
