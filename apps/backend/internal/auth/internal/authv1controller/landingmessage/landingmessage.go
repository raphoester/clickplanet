package landingmessage

import (
	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
)

func LandingOf(landing *authv1.Landing) arrivals.Landing {
	// Anybody can type an address: an id that is not an account names no recruiter, and makes no call fail.
	recruiter, err := accounts.AccountIDOf(landing.GetRecruiterAccountId())
	if err != nil {
		recruiter = accounts.AccountID{}
	}
	return arrivals.Landing{Tag: landing.GetTag(), Referrer: landing.GetReferrer(), Recruiter: recruiter}
}
