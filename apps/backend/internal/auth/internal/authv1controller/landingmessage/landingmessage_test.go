package landingmessage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/landingmessage"
)

var recruiter = accounts.AccountID{0: 0x01, 6: 0x70, 8: 0x80, 15: 0x77}

func TestTheLandingIsWhatTheBrowserSaid(t *testing.T) {
	landing := landingmessage.LandingOf(&authv1.Landing{
		Tag: "rc", Referrer: "discord.com", RecruiterAccountId: proto.String(recruiter.String()),
	})

	assert.Equal(t, arrivals.Landing{Tag: "rc", Referrer: "discord.com", Recruiter: recruiter}, landing)
}

func TestALandingWithNoRecruiterNamesNone(t *testing.T) {
	for name, landing := range map[string]*authv1.Landing{
		"no landing":   nil,
		"no recruiter": {Tag: "rc"},
		"an empty one": {Tag: "rc", RecruiterAccountId: proto.String("")},
		"not an id":    {Tag: "rc", RecruiterAccountId: proto.String("kutza")},
		"the nil id":   {Tag: "rc", RecruiterAccountId: proto.String("00000000-0000-0000-0000-000000000000")},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, arrivals.AccountID{}, landingmessage.LandingOf(landing).Recruiter)
		})
	}
}
