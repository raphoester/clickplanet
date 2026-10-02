package rpc_account_reader

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

const askTimeout = 2 * time.Second

type Reader struct {
	dial Dialer
}

func New(dial Dialer) *Reader {
	return &Reader{dial: dial}
}

func (r *Reader) Linked(ctx context.Context, account players.AccountID) (bool, error) {
	res, err := r.account(ctx, account)
	if err != nil {
		return false, fmt.Errorf("failed to ask auth whether the account is linked: %w", err)
	}
	return res.GetLinked(), nil
}

func (r *Reader) CreatedAt(ctx context.Context, account players.AccountID) (time.Time, error) {
	res, err := r.account(ctx, account)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to ask auth when the account was made: %w", err)
	}
	if res.GetCreatedAtUnixMs() == 0 {
		return time.Time{}, nil
	}
	return time.UnixMilli(res.GetCreatedAtUnixMs()).UTC(), nil
}

func (r *Reader) CreationDates(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]time.Time, error) {
	dates := make(map[players.AccountID]time.Time, len(accounts))
	if len(accounts) == 0 {
		return dates, nil
	}

	client, baseURL, err := r.dial.Dial()
	if err != nil {
		return nil, fmt.Errorf("failed to reach the auth module: %w", err)
	}

	ids := make([]string, len(accounts))
	for i, account := range accounts {
		ids[i] = account.String()
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := authv1connect.NewInternalServiceClient(client, baseURL).
		GetCreationDates(ctx, connect.NewRequest(&authv1.GetCreationDatesRequest{AccountIds: ids}))
	if err != nil {
		return nil, fmt.Errorf("failed to call auth.v1.InternalService/GetCreationDates: %w", err)
	}

	for _, date := range res.Msg.GetDates() {
		account, err := players.AccountIDOf(date.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("auth answered a creation date for %w", err)
		}
		dates[account] = time.UnixMilli(date.GetCreatedAtUnixMs()).UTC()
	}
	return dates, nil
}

func (r *Reader) account(ctx context.Context, account players.AccountID) (*authv1.GetAccountResponse, error) {
	client, baseURL, err := r.dial.Dial()
	if err != nil {
		return nil, fmt.Errorf("failed to reach the auth module: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := authv1connect.NewInternalServiceClient(client, baseURL).
		GetAccount(ctx, connect.NewRequest(&authv1.GetAccountRequest{AccountId: account.String()}))
	if err != nil {
		return nil, fmt.Errorf("failed to call auth.v1.InternalService/GetAccount: %w", err)
	}
	return res.Msg, nil
}
