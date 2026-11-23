//go:build js && wasm

package main

import (
	"context"
	"net/http"

	"github.com/syumai/workers-go/cloudflare"
	"github.com/syumai/workers-go/cloudflare/fetch"
	"github.com/syumai/workers-go/exp/cloudflare/email"
)

func main() {
	email.Handle(func(message *email.ForwardableEmailMessage) error {
		return Backend{
			URL:      cloudflare.Getenv("MAILROOM_URL"),
			Secret:   cloudflare.Getenv("MAILROOM_SECRET"),
			Operator: cloudflare.Getenv("OPERATOR_ADDRESS"),
			HTTP:     fetch.NewClient().HTTPClient(fetch.RedirectModeFollow),
		}.Deliver(context.Background(), forwardable{message})
	})
}

type forwardable struct {
	*email.ForwardableEmailMessage
}

func (m forwardable) Forward(to string) error {
	_, err := m.ForwardableEmailMessage.Forward(to, http.Header{})
	return err
}
