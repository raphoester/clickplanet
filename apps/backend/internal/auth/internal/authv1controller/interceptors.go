package authv1controller

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
)

type MintLimiter = cpconnect.Limiter

var (
	ErrTooManySessions = errors.New("too many session attempts")
	ErrTooManyResumes  = errors.New("too many session resumes")
)

// Apart from the mint's: a resume costs no Turnstile check, and every page load makes one.
func NewResumeRateLimitInterceptor(limiter MintLimiter) connect.Interceptor {
	return cpconnect.NewRateLimitInterceptor(limiter, ErrTooManyResumes, authv1connect.AuthServiceResumeSessionProcedure)
}

func NewRateLimitInterceptor(limiter MintLimiter) connect.Interceptor {
	return cpconnect.NewRateLimitInterceptor(
		limiter, ErrTooManySessions,
		authv1connect.AuthServiceCreateSessionProcedure,
		authv1connect.AuthServiceStartSignInProcedure,
		authv1connect.AuthServiceCompleteSignInProcedure,
		authv1connect.AuthServiceStartEmailSignInProcedure,
		authv1connect.AuthServiceCompleteEmailSignInProcedure,
	)
}
