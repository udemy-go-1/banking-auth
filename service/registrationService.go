package service

import (
	"fmt"
	"github.com/udemy-go-1/banking-auth/domain"
	"github.com/udemy-go-1/banking-auth/dto"
	"github.com/udemy-go-1/banking-lib/errs"
	"net/url"
	"os"
	"time"
)

type RegistrationService interface { //service (primary port)
	Register(dto.RegistrationRequest) (*dto.RegistrationResponse, *errs.AppError)
	CheckRegistration(string) (bool, *errs.AppError)
	ResendLink(dto.ResendRequest) *errs.AppError
	FinishRegistration(string) *errs.AppError
}

type DefaultRegistrationService struct { //business/domain object
	registrationRepo domain.RegistrationRepository
	emailRepo        domain.EmailRepository
}

func NewRegistrationService(regRepo domain.RegistrationRepository, emailRepo domain.EmailRepository) DefaultRegistrationService {
	return DefaultRegistrationService{regRepo, emailRepo}
}

// Register uses the given dto.RegistrationRequest to check whether any of the following cases are true:
//
// 1) the given email was already used to register for an app account
//
// 2) there is already a User with the given username, or
//
// 3) there is already a Customer with the given email.
//
// If so, the request is rejected. Otherwise, it is saved to the db, and a one-time use JWT is generated to form
// a confirmation link which is then emailed to the requester.
func (s DefaultRegistrationService) Register(request dto.RegistrationRequest) (*dto.RegistrationResponse, *errs.AppError) {
	registration := domain.NewRegistration(request)

	if appErr := s.registrationRepo.IsEmailUsed(registration.Email); appErr != nil {
		return nil, appErr
	}
	if appErr := s.registrationRepo.IsUsernameTaken(registration.Username); appErr != nil {
		return nil, appErr
	}

	if err := s.registrationRepo.Save(registration); err != nil {
		return nil, err
	}

	if err := s.createAndSendLink(registration); err != nil {
		return nil, err
	}

	return registration.ToDTO(), nil
}

func buildConfirmationURL(ott string) string {
	addr := os.Getenv("FRONTEND_SERVER_ADDRESS")
	port := os.Getenv("FRONTEND_SERVER_PORT")
	u := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", addr, port),
		Path:   "register/check",
	}

	v := url.Values{}
	v.Add("ott", ott)
	u.RawQuery = v.Encode()

	return u.String()
}

func (s DefaultRegistrationService) createAndSendLink(reg domain.Registration) *errs.AppError {
	ott, err := reg.GenerateOneTimeToken()
	if err != nil {
		return err
	}
	link := buildConfirmationURL(ott)

	timeEmailed, err := s.emailRepo.SendConfirmationEmail(reg.Email, link)
	if err != nil {
		return err
	}

	if err = s.registrationRepo.UpdateLastEmailedInfo(reg, timeEmailed); err != nil {
		return err
	}

	return nil
}

// CheckRegistration checks that the given token is valid, tries to retrieve an existing Registration from the token
// claims, then returns its status.
func (s DefaultRegistrationService) CheckRegistration(tokenString string) (bool, *errs.AppError) {
	claims, err := domain.ValidateOneTimeToken(tokenString, false)
	if err != nil {
		return false, err
	}

	registration, err := s.registrationRepo.FindFromEmail(claims.Email)
	if err != nil {
		return false, err
	}

	return registration.IsConfirmed(), nil
}

// ResendLink retrieves the recipient's email from the token claims if needed, tries to retrieve an existing
// Registration from the email and checks if resending the confirmation link to this email is allowed before doing so.
func (s DefaultRegistrationService) ResendLink(request dto.ResendRequest) *errs.AppError {
	var email string
	if request.Type == dto.ResendRequestTypeUsingToken {
		claims, err := domain.ValidateOneTimeToken(request.TokenString, true)
		if err != nil {
			return err
		}
		email = claims.Email
	} else if request.Type == dto.ResendRequestTypeUsingEmail {
		email = request.Email
	}

	registration, err := s.registrationRepo.FindFromEmail(email)
	if err != nil {
		return err
	}

	if err = registration.CanResendEmail(); err != nil {
		return err
	}

	if err = s.createAndSendLink(*registration); err != nil {
		return err
	}

	return nil
}

// FinishRegistration double-checks that the given token is valid, retrieves an existing Registration from the token
// claims, initializes the new user, and fills the remaining fields of the Registration and uses it to update the db.
func (s DefaultRegistrationService) FinishRegistration(tokenString string) *errs.AppError {
	claims, err := domain.ValidateOneTimeToken(tokenString, false)
	if err != nil {
		return err
	}

	registration, err := s.registrationRepo.FindFromEmail(claims.Email)
	if err != nil {
		return err
	}

	createTime := time.Now().Format(domain.FormatDateTime)
	customerId, err := s.registrationRepo.CreateNecessaryAccounts(registration, createTime)
	if err != nil {
		return err
	}

	completedRegistration := registration.Confirm(customerId, createTime)
	if err = s.registrationRepo.Update(completedRegistration); err != nil {
		return err
	}

	return nil
}
