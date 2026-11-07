// Package moderationdata holds the chat moderation policy, copied from the monorepo's /docs/moderation.md by
// `make moderation`: the backend builds from its own directory, and the chat's judge sends the policy to Claude with
// every report it judges, so a change to the doc changes what the judge does once it is copied here.
//
// Embedded rather than read from disk, for the reason generated/quiz is: cmd/api is a self-contained container.
package moderationdata

import _ "embed"

//go:embed moderation.md
var policy string

func Policy() string { return policy }
