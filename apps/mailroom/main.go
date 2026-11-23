//go:build js && wasm

package main

import (
	"context"

	"github.com/syumai/workers-go/cloudflare"
	"github.com/syumai/workers-go/cloudflare/fetch"
	"github.com/syumai/workers-go/exp/cloudflare/email"
)

func main() {
	email.Handle(func(message *email.ForwardableEmailMessage) error {
		return Backend{
			URL:    cloudflare.Getenv("MAILROOM_URL"),
			Secret: cloudflare.Getenv("MAILROOM_SECRET"),
			HTTP:   fetch.NewClient().HTTPClient(fetch.RedirectModeFollow),
		}.Deliver(context.Background(), message)
	})
}
