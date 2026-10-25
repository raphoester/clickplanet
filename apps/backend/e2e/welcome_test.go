package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
)

func TestANewPlayerGetsASpreadAtOnceAndAQuizWhenItsClicksRunOut(t *testing.T) {
	const bank = 3

	game := startGameWith(t, func(configs stackConfigs) {
		configs.planet.RateLimiter.Burst = bank
		configs.planet.RateLimiter.PerSecond = 0.001
		configs.planet.Bonus.Quiz.Enabled = true
		configs.planet.Bonus.Welcome.Within = 10 * time.Minute
	})
	ada := game.newPlayer(t)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	listen := connect.NewRequest(&planetv1.ListenForEventsRequest{})
	ada.send(listen.Header())
	stream, err := planetv1connect.NewClickServiceClient(http.DefaultClient, game.baseURL).ListenForEvents(ctx, listen)
	require.NoError(t, err)
	require.True(t, stream.Receive(), "the stream opens with a heartbeat")

	boxes := make(chan *planetv1.BonusOffered, 4)
	quizzes := make(chan *planetv1.QuizOffered, 4)
	go func() {
		for stream.Receive() {
			if box := stream.Msg().GetBonusOffered(); box != nil {
				boxes <- box
			}
			if quiz := stream.Msg().GetQuizOffered(); quiz != nil {
				quizzes <- quiz
			}
		}
	}()

	ada.click(1, "fr")
	select {
	case box := <-boxes:
		assert.Equal(t, planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS, box.GetKind())
	case <-time.After(5 * time.Second):
		t.Fatal("no box for the new player")
	}

	for tile := uint32(2); tile <= bank; tile++ {
		select {
		case quiz := <-quizzes:
			t.Fatalf("a quiz before the bank ran out: %+v", quiz)
		default:
		}
		ada.click(tile, "fr")
	}

	select {
	case quiz := <-quizzes:
		assert.NotEmpty(t, quiz.GetToken())
	case <-time.After(5 * time.Second):
		t.Fatal("no quiz once the bank ran out")
	}
}
