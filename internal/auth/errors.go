package auth

import (
	"errors"
	"time"
)

var (
	ErrInvalidCredentials   = errors.New("auth: invalid credentials")
	ErrAccountBlocked       = errors.New("auth: account blocked")
	ErrAccountSuspended     = errors.New("auth: account suspended")
	ErrEmailNotVerified     = errors.New("auth: email not verified")
	ErrMFARequired          = errors.New("auth: MFA required")
	ErrMFANotEnrolled       = errors.New("auth: MFA not enrolled")
	ErrInvalidToken         = errors.New("auth: invalid token")
	ErrTokenExpired         = errors.New("auth: token expired")
	ErrTokenUsed            = errors.New("auth: token already used")
	ErrTokenInvalid         = errors.New("auth: token invalid")
	ErrInvalidPurpose       = errors.New("auth: invalid token purpose")
	ErrReauthRequired       = errors.New("auth: reauthentication required")
	ErrPrivilegeEscalation  = errors.New("auth: privilege escalation blocked")
	ErrWebAuthnNotConfigured = errors.New("auth: webauthn not configured")
	ErrNoEncryptionKey      = errors.New("auth: no encryption key available")
	ErrSessionNotFound      = errors.New("auth: session not found")
	ErrSessionRevoked       = errors.New("auth: session revoked")
	ErrSessionExpired       = errors.New("auth: session expired")
	ErrUserInactive         = errors.New("auth: user inactive")
	ErrMaxSessionsReached   = errors.New("auth: max sessions reached")
	ErrEmailExists          = errors.New("auth: email already exists")
	ErrUsernameExists       = errors.New("auth: username already exists")
	ErrUserNotFound         = errors.New("auth: user not found")
	ErrPasswordTooShort     = errors.New("auth: password too short (minimum 12 characters)")
	ErrPasswordTooLong      = errors.New("auth: password too long (maximum 256 bytes)")
	ErrCommonPassword       = errors.New("auth: password is too common")
	ErrPasswordContainsID   = errors.New("auth: password contains email or username")
	ErrHashBusy             = errors.New("auth: hasher busy")
	ErrInvalidHashFormat    = errors.New("auth: invalid hash format")
	ErrUnsupportedVersion   = errors.New("auth: unsupported argon2 version")
	ErrParamsOutOfBounds    = errors.New("auth: argon2 params out of bounds")
	ErrTOTPNotEnrolled      = errors.New("auth: TOTP not enrolled")
	ErrTOTPNotConfirmed     = errors.New("auth: TOTP not confirmed")
	ErrInvalidTOTPCode      = errors.New("auth: invalid TOTP code")
	ErrCredentialNotFound   = errors.New("auth: credential not found")
	ErrSignCountInvalid     = errors.New("auth: invalid sign count (possible cloning)")
)

type ErrRateLimited struct {
	RetryAfter time.Duration
}

func (e ErrRateLimited) Error() string {
	return "rate limited"
}