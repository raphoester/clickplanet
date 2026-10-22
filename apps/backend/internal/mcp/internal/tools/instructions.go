package tools

// What Claude is told once it connects, before any tool's own description.
const Instructions = `ClickPlanet (clickplanet.lol) is a game where players click the tiles of a globe to paint them with a
country's flag. These are its operator's tools, on the production server, as the operator who connected you: what a
tool changes, it changes for every player at once.

- A tool that only reads runs without asking. Before any other, say what it will change; where it has dry_run, run it
  with dry_run first and show what it would do.
- A player is named by its account_id, and by its scope: its address over IPv4, its /64 over IPv6. Many players can
  share a scope, so prefer the account_id when acting on one player.
- The ledger behind FindPlayers, TopPlayers and RevertPlayer holds the last 72 hours of play.
- A country id is ISO 3166-1 alpha-2 in lower case: "fr". A duration is in seconds: "3600s". A time is RFC 3339.
- A refused call says why, as a Connect code and a sentence: read it before trying again.`
