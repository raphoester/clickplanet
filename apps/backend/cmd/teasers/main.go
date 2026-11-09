// The teasers job: each battle posted about a day before it starts, its cards as a photo carousel on TikTok and a
// slideshow on YouTube, through the server's publishing module. The workflow runs due, renders each teaser it lists
// with the frontend's npm run teaser, then runs post.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo"
)

const usage = `usage:
  teasers due -out <folder> [-ssh <user@host>]
  teasers post -teasers <folder> [-ssh <user@host>] [-summary <file>]

Both ask the production backend over ssh, as its operator does: CLICKPLANET_SSH stands in for -ssh. due lists the
battles starting within 36 hours that no post teases yet, writes each one's teaser.json in a folder of its own under
-out, for the frontend's npm run teaser to render beside it, and prints the folders' names as a JSON array. post reads
every folder under -teasers the renderer filled, lists the battles again, and posts each one still due through the
backend's publishing module, at the first slot its calendar gives on TikTok and YouTube both by an hour before the
battle, else on TikTok alone, else on YouTube alone; it adds what became of each to -summary, default the standard
output, and fails when one failed.`

const hostVariable = "CLICKPLANET_SSH"

var (
	errArguments = errors.New("wrong arguments")
	errFailed    = errors.New("a teaser was not posted")
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout)
	stop()

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return misused(errArguments)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))

	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var config promo.JobConfig
	flags.StringVar(&config.SSH, "ssh", os.Getenv(hostVariable), "user@host of the machine the backend runs on")
	switch args[0] {
	case "due":
		folder := flags.String("out", "", "the folder each teaser's folder is made in")
		if err := flags.Parse(args[1:]); err != nil || *folder == "" || config.SSH == "" {
			return misused(errors.Join(errArguments, err))
		}
		return due(ctx, *folder, config, out, logger)
	case "post":
		folder := flags.String("teasers", "", "the folder the teasers were rendered in")
		summary := flags.String("summary", "", "the file the summary of the run is added to, default the standard output")
		if err := flags.Parse(args[1:]); err != nil || *folder == "" || config.SSH == "" {
			return misused(errors.Join(errArguments, err))
		}
		return post(ctx, *folder, *summary, config, out, logger)
	default:
		return misused(errArguments)
	}
}

func misused(err error) error {
	return fmt.Errorf("%w\n%s", err, usage)
}

// The folders' names, the workflow's matrix: [] for none.
func due(ctx context.Context, folder string, config promo.JobConfig, out io.Writer, logger *slog.Logger) error {
	listed, err := promo.ListDue(ctx, folder, config, logger)
	if err != nil {
		return fmt.Errorf("failed to list the battles due their teaser: %w", err)
	}
	names := make([]string, 0, len(listed))
	for _, one := range listed {
		names = append(names, one.Folder)
	}
	written, err := json.Marshal(names)
	if err != nil {
		return fmt.Errorf("failed to write the folders: %w", err)
	}
	if _, err := fmt.Fprintf(out, "%s\n", written); err != nil {
		return fmt.Errorf("failed to print the folders: %w", err)
	}
	return nil
}

func post(ctx context.Context, folder, summary string, config promo.JobConfig, out io.Writer, logger *slog.Logger) error {
	report, err := promo.PostTeasers(ctx, folder, config, logger)
	if err != nil {
		return fmt.Errorf("failed to post the teasers: %w", err)
	}

	to := out
	if summary != "" {
		file, err := os.OpenFile(summary, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // the operator names the file.
		if err != nil {
			return fmt.Errorf("failed to open the summary: %w", err)
		}
		defer func() { _ = file.Close() }()
		to = file
	}
	if _, err := io.WriteString(to, summaryOf(report)); err != nil {
		return fmt.Errorf("failed to write the summary: %w", err)
	}
	if report.Failed() {
		return errFailed
	}
	return nil
}

// In Markdown, for the run's page.
func summaryOf(report promo.TeasersReport) string {
	var summary strings.Builder
	summary.WriteString("## Battle teasers\n\n")
	if len(report) == 0 {
		summary.WriteString("No teaser to post this run.\n")
		return summary.String()
	}
	for _, outcome := range report {
		fmt.Fprintf(&summary, "- **%s**: ", outcome.Title)
		if outcome.Result != promo.TeaserScheduled {
			fmt.Fprintf(&summary, "%s, %s\n", outcome.Result, outcome.Why)
			continue
		}
		platforms := make([]string, 0, len(outcome.Platforms))
		for _, platform := range outcome.Platforms {
			platforms = append(platforms, string(platform))
		}
		fmt.Fprintf(&summary, "scheduled at %s on %s (post %s)", outcome.At.UTC().Format("Mon 2 Jan 15:04 UTC"),
			strings.Join(platforms, " and "), outcome.Post)
		refused := make([]string, 0, len(outcome.Refused))
		for _, platform := range outcome.Platforms {
			if why, ok := outcome.Refused[platform]; ok {
				refused = append(refused, fmt.Sprintf("%s refused it: %s", platform, why))
			}
		}
		slices.Sort(refused)
		if len(refused) > 0 {
			fmt.Fprintf(&summary, "; %s", strings.Join(refused, "; "))
		}
		summary.WriteString("\n")
	}
	return summary.String()
}
