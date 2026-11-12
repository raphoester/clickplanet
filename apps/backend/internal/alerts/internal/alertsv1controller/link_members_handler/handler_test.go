package link_members_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	alertsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/alertsv1controller/link_members_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links/usecases/link_members_usecase"
)

type stubUseCase struct {
	out link_members_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context) (link_members_usecase.Out, error) {
	return s.out, s.err
}

func TestTheRunIsCounted(t *testing.T) {
	useCase := stubUseCase{out: link_members_usecase.Out{
		Linked: 7, Named: 5, Outside: 13, Failed: []link_members_usecase.Failure{{User: "4242", Err: errors.New("discord answered 500")}},
	}}

	res, err := link_members_handler.New(useCase).LinkMembers(t.Context(), connect.NewRequest(&alertsv1.LinkMembersRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&alertsv1.LinkMembersResponse{Linked: 7, Outside: 13, Failed: 1, Named: 5}, res.Msg), "got %v", res.Msg)
}
