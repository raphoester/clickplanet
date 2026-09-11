package main

import (
	"context"
	"fmt"
	"os"

	"github.com/raphoester/clickplanet.lol-backend/internal/app"
)

func main() {
	if err := app.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
