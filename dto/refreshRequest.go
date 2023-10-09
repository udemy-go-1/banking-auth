package dto

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/udemy-go-1/banking-auth/domain"
	"github.com/udemy-go-1/banking-lib/errs"
	"github.com/udemy-go-1/banking-lib/logger"
	"time"
)

type RefreshRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// ValidateAccessToken checks that the given access token is valid (signed by this app) and has already expired.
func (r RefreshRequest) ValidateAccessToken() (*jwt.Token, *errs.AppError) {
	var validatedAccessToken *jwt.Token
	var appErr *errs.AppError
	var isAccessTokenExpired bool
	if validatedAccessToken, appErr = domain.GetValidAccessTokenFrom(r.AccessToken, true); appErr != nil {
		return nil, appErr
	}

	isAccessTokenExpired, appErr = isExpired(validatedAccessToken)
	if appErr != nil {
		return nil, appErr
	}
	if !isAccessTokenExpired {
		logger.Error("Access token not expired yet")
		return nil, errs.NewAuthenticationError("Cannot generate new access token until current one expires")
	}

	return validatedAccessToken, nil
}

func isExpired(token *jwt.Token) (bool, *errs.AppError) {
	date, err := token.Claims.GetExpirationTime() //registered claims "exp", etc
	if err != nil {
		logger.Error("Error while checking token's expiry time: " + err.Error())
		return false, errs.NewUnexpectedError(err.Error())
	}
	if !date.Time.After(time.Now()) { //token expiry date is before or at current time = expired
		return true, nil
	}
	return false, nil
}