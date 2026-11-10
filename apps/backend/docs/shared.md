# Shared (`internal/shared/`)

Shared infrastructure: `cpbootstrap` (the composite layer), `cpredis` (the event bus over Redis Streams, which each module connects to on its own — see [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened)), `cpatomic` (the `Transactioner` and `Outbox` ports a use case takes — see [Transactions](transactions.md)), `cpoutbox` (the outbox a use case records in, inside its transaction, and its relay — see [What may not be lost](composite-layer.md#what-may-not-be-lost-the-outbox)), `cpcountries`, `cpscripts` (the historic scripts chat and names refuse), `cpphrases` (a listed phrase found however it is spelled, for the chat's banned phrases and player's banned names), `cpconfigs` (YAML + env config via koanf), `cphttpserver` (middleware), `cpprom` (Prometheus), `cptime`, `cpctx` (what a request carries: the caller's address, its account, and `Operator`, who calls an admin service, which auth's admin gate puts there and chat and player keep with what an operator did), `cpconnect`, `cpratelimit`, `cpipblock`, `cpipscope`, `cpsession`, `cpsessionverifier`, `cpsecrets`, `cppg` (postgres: a store reaches it only through `db.On(ctx)` — see [Transactions](transactions.md) and [Durability](durability.md)), `cpcolls` (collections), `cpzernio` (the client of the posting service), `cpanthropic` (the client of Anthropic's API), `cpproto` (`Overlay`: one proto message's values written over another's, which planet's and the antibot's rule sets play over `default` with).

**Every package here is prefixed `cp`, and a new one must be.** A call site reads
`cptime.SystemClock{}` or `cpctx.GetSourceIP(ctx)`, so the prefix says the
dependency is this layer's without the reader going to the import block — and an
unprefixed name in a module is a module's own package by construction. It also
settles the collisions a shared layer attracts: `cptime` beside stdlib `time`,
`cpsession` beside the session *module*, `cpconnect` beside `connectrpc.com/connect`.
`cpsession` is what let the old `internal/session/module.go` drop the `sharedsession`
import alias it needed while the two were both called `session`.

**There is no logging package here, deliberately.** Every constructor that logs
takes a `*slog.Logger` from the standard library. The wrapper that used to sit
here was an interface of four methods over `log/slog` plus a `cplf` field type
that stringified every value on the way in — so a structured logger was being
flattened to strings by the layer whose job was to keep them structured. A
handler is the supported way to change where logs go, and `slog.New(slog.DiscardHandler)`
is the nop. The one thing lost is the name `Warning`, which is `Warn` in slog.

The `util` suffix is dropped rather than prefixed — `cpctx`, not `cpctxutil`.
A package named for what it *is* stays that; one named "utilities for X" was
never saying anything the prefix does not.

`cpbootstrap` runs the modules — see [The composite layer](composite-layer.md). It knows nothing about this game: it takes a list of modules, builds each one under a startup deadline, mounts what they claimed on one router, and serves until the process is signalled. The one thing it builds for them is the click token's check (`props.Sessions`, a `cpsessionverifier`), since the token's format is the shared kernel every module verifies the same way.

Two of these are here because both bounded contexts need them and neither should depend on the other:

- `cpsecrets` — the random hex a config may leave it to the server to invent. The player module's tag salt is the only one left, and it costs nothing: the tag never leaves memory. `cpbootstrap` also draws the boot id with it at each start (see [A server restart reloads the page](../../../docs/reload-on-restart.md)). The click token's key is not invented, because a key the server made up could not verify what another restart had minted — see [Sessions](sessions.md).

`cpsession` mints and verifies the click token — see [Sessions](sessions.md). It is here because **both** contexts read it: `auth` mints with its `Signer`, `planet` verifies with its `Verifier`, and neither may depend on the other. The two halves are separate types over separate config, so what each context can do with it is decided at compile time.

The siteverify client it is fed by is **not** here. `turnstile` sat here on the same "both contexts need it" rule, but only one ever did, so it now lives at `session/internal/turnstile` where the compiler keeps it. **The bar is not that a package is shareable, it is that it would read the same in any other program and that two modules actually import it** — "shared" names the symptom, and a directory admitted on the weaker reading becomes a dumping ground. `cpsecrets` passes narrowly — the player module and `cpbootstrap` are its only callers today, but it is twenty lines of `crypto/rand` with no domain in it at all.

`cpscripts.Historic` says if a character is of a historic script, one no one writes today: the code points whose UTS #39 `Identifier_Type` is `Exclusion` (cuneiform, Egyptian hieroglyphs, runes, Gothic, Linear B…). Chat refuses a text and player a name with one. `make historic-scripts` fetches Unicode's `IdentifierType.txt` and keeps only those lines in `cpscripts/historic.txt`, embedded; a file that does not parse, is out of order or holds under 300 ranges stops the process at start. `Obsolete` is left out on purpose: it holds letters people still type, like the Arabic tatweel (ـ). `Limited_Use` scripts (Tifinagh, Yi, N'Ko…) are living, and pass.

`cpphrases` finds a listed phrase in a text however it is spelled, for the chat's banned phrases and player's banned
names (see [Chat](chat.md) and [Player](player.md)). It knows no list: each context keeps its own, and says where a
phrase may begin, `cpphrases.New(match, phrases...)`, then `List.Said(text)` answers the phrase as it was listed.
- **Both sides are folded the same way**: decomposed (NFKD: a fullwidth, circled or mathematical letter is its letter),
  with no accent, in lower case, a Cyrillic or Greek look-alike as the Latin letter it passes for (`lookAlikes`, with
  the small capitals of the "fancy text" generators and the regional indicators a flag is made of), and no separator:
  a space, a punctuation mark, a symbol or a zero-width character between two letters is nothing.
- **Leetspeak folds the text alone**: a digit or a sign typed reads as its letter (`0 3 4 5 7 8 9 @ $ €`; `1`, `|` and
  `!` as an i or an l), and a digit listed in a phrase is that digit. So "n1gg3r" says "nigger", "14 88" says "1488",
  and "Koala88" does not, though its l reads like a 1.
- **Each letter of a phrase takes one or more of itself**, so "niiigggerr" says "nigger" and "Niger" does not: the
  phrase's double g needs two. Never a collapse of the text's repeated letters, which would make the two one.
- **Where a phrase of letters may begin and end** is the caller's match:
  - `InWords`, for sentences. A phrase begins where a word does (after a separator, or a letter after a digit or a
    leet sign), so "snigger" passes. A phrase of one word of letters takes any ending, as a slur does ("niggers"). Any
    other, of many words or with a digit, is whole and ends where a word does: the next letter starts a new word, or
    is not of the phrase's kind (a letter after a letter, a digit after a digit). So "white power" lets "the white
    powers of Europe" pass and still finds "WHITE POWER!!!", and "1488" lets "14880" pass.
  - `Anywhere`, for a name, which is one word: a phrase begins and ends anywhere, so "xXHitlerXx" and "WhitePowerful"
    are found.

  A phrase with no cased letter or digit (卐) matches anywhere either way, and one that folds to nothing is left out,
  since it would match every text.
- Its tests pin each folding and both matches; each caller's tests pin its own list.

`cpsessionverifier` is the exception to that bar: it knows `auth.v1.InternalService`, so it would not read the same in another program. It is the `cpconnect.SessionVerifier` that `cpbootstrap` builds once per process and hands every module as `props.Sessions`, which takes the key from `auth` once — see [Sessions](sessions.md). It is here because the alternative was one copy in each module, and a fix to one copy missed the others. Until 2026-11-04 `planet`, `player`, `progression`, `chat`, `scoring`, `support`, `missions`, `alerts` and `clips` each built one.

`cpcountries` is the ISO country list both the tile game and the chat validate against. `cpipblock` is the VPN prefix set — see [VPN blocklist](vpn-blocklist.md). `cpratelimit` is a keyed token bucket held in this process, like the tile map it protects — with one API instance, a shared counter would buy nothing. Its `Run` loop periodically forgets the buckets that have refilled to capacity, which is free: such a bucket holds exactly what a freshly created one would, and without it the map would keep an entry per address that ever clicked.

`cpipscope` decides what a scope's bucket is keyed on, and every throttle goes through it. Over IPv4 that is the address; over IPv6 it is the surrounding **/64**, because the smallest allocation a subscriber receives is a /64 and most receive far more — a bucket per v6 address is one the same line walks out of by picking its next address, turning one home connection into thousands of callers with a throttle each. The session token binds to the same unit, so the address a token is valid for and the address that spends a budget cannot diverge. Blocking deliberately does **not** use it: the VPN and datacenter lists are precise prefixes already, and widening a hit to the surrounding /64 would refuse neighbours who are not on them.
`cpzernio` is Zernio's HTTP API as Go: a generated client, the vendored OpenAPI document it is generated from, and the two files that configure the generation, all in the one directory, as `cpipblock` keeps its lists in `cpdata`. **Nothing in it is written by hand but the two config files**: `make zernio-spec` refreshes `openapi.json`, `make zernio` writes `client.gen.go`. It knows nothing of clips or of this game, which is why it is here and not in the module: the server's publishing module (`posts/zernio`, which posts) and its `stats/zernio_source` (which reads the posts' numbers) are its callers, and hold everything that is ours (see [Publishing](publishing.md) and [The clips job](clips-job.md)). The linters, gofumpt and `make deadcode` leave `client.gen.go` alone by its generated header, not by its path.
`cpanthropic` builds the client of Anthropic's API (`NewClient`, from a key and, for a test, a base URL; no key is
`ErrNoKey`) and says what a failed call went wrong with, the request left out (`Redacted`): the SDK's error dumps
the request it failed on, key and all. **It knows no model, no prompt and no timeout**: those are each caller's, set
per request, so one client serves calls of any length. The clips job's editor, the clips module's name judge
(`claude_name_judge`, see [The names a post may show](clip-name-verdicts.md)) and the chat's judge (see
[The judge](chat.md#the-judge)) are its callers today.
`cps3` is an S3-compatible bucket, in production Cloudflare R2: links signed in their query (AWS Signature Version 4,
no SDK; `signature_test.go` signs the S3 reference's own example), to upload one object of an exact type and size, to
view it, or to download it under a name (`response-content-disposition`), and a `HEAD` and a `DELETE` of its own. **It
knows keys, not files**: the support module keeps its notes' files at the root (`s3_attachment_bucket`), the clips
module its videos under `clips/videos/` (`s3_video_bucket`), the promo module its backgrounds under
`promo/backgrounds/` (`s3_background_bucket`), each with its own config block, which may name the same bucket. Its tests run on a real S3 server, the Versity gateway in a container.
`cpcolls` holds the collections the standard library does not — today `Set[T]`. **A set is a `*cpcolls.Set`, never a map.** A `map[T]struct{}` or a `map[T]bool` written `m[k] = true` fails `make lint`: a ruleguard rule in `tools/ruleguard/rules.go`, run by gocritic, reports both everywhere but in `cpcolls` itself. A nil `*Set` reads as empty, like a nil map, so a lookup in a map of sets needs no `ok` check.
