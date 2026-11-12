# Pages and routes

Five pages, built by Vite as a multi-page app (`build.rollupOptions.input` in
`vite.config.ts`):

| Path | File | What |
|---|---|---|
| `/` | `index.html` | The home page. Plain HTML, and one small module for the season's countdown and the next battle's. |
| `/play` | `play.html` | The game. |
| `/archive` | `archive.html` | The map of the seasons that ended (see [A season's map](season-map.md)). The `archiveRoute` plugin serves it at that path in `npm run dev`. |
| `/auth/callback` | `auth/callback.html` | The game again, for the sign-in callback. |
| `/privacy`, `/terms` | `privacy.html`, `terms.html` | Plain pages, no bundle. |
| `/<flag>`, `/<flag>/<name>` | `index.html` | A player's address: its first script opens the game on the flag. |
| anything else | `index.html` | `not_found_handling` fallback. |

**Why `/` is not the game.** Google's brand verification, which lets anyone
sign in with Google, was refused twice: the checker runs JavaScript, saw only
the globe, and found no text that says what the app is and no visible link to
the privacy policy. A plain intro inside `#root` did not help: hidden as soon as
JavaScript ran (it flashed), the checker never saw it. So `/` is a plain page
built like `privacy.html`: what the game is, "no account needed", a Play button,
and links to both pages, the contact address and Discord. **Keep that text and
those links on it**; they are what the verification reads.

**Only a first visit sees it.** An inline script, first in the home page's
`<head>`, sends a browser that holds `COUNTRY_STORAGE_KEY` in local storage to
`/play` with `location.replace`, before anything is painted. The game writes
that key on its first render, so a browser that opened the game once never
sees the home page again. `location.search` and `location.hash` go along, so
an old share link such as `/?f=de` still reaches the game with its query. Crawlers
have no storage and read the home page. `homePage.test.ts` pins the key in the
script to the constant. **`/#home` does not redirect**: it is the "Home page"
link at the bottom of the About modal.

**A flag's address opens the game, for a first visit too.** Every share a
player makes is `clickplanet.lol/<flag>/<name>`, or `clickplanet.lol/<flag>`
for a guest (see [Sharing the globe](sharing.md)). No file is at that path, so
the fallback serves the home page, and its first script, before anything is
painted, sends every browser to `/play?f=<flag>&s=rc`, with `&r=<name>` when
the address has a name: the same query an old share link carried, which the
game already reads (`sharedCountry`, and the landing's tag and recruiter, see
[Arrivals](arrivals.md)). The flag and the name win over the same words in the
query; the rest of the query and the hash go along. Only a flag the game knows
counts, in any case: the build writes every code of `countries.json` into the
script (`withFlagCodes`, the `flagAddresses` plugin in `vite.config.ts`), so
`/zz`, a third segment, or `/pl/kutza/more` is any other path, as before. A
name that is not one (`%E0%A4%A`) is dropped and the game still opens on the
flag. The script keeps the page before in session storage for the game, since
`location.replace` makes the address itself the game's referrer. The home page
stays at `/`, for Google's check; crawlers and link previews of an address read
it too, as before. `homePage.test.ts` runs the script on addresses.

**The CrazyGames page never shows it.** On `crazygames.clickplanet.lol` the same
script sends every visitor to `/play`, with no storage read and `#home` too, so
`/`, `/play/` and every other path that falls back to the home page reach the
game, with no flag read from the path. A reviewer comes with no storage: on Oct 30 2026, the day the game was
sent to CrazyGames, four outside visitors there got the home page and never
pressed Play, and CrazyGames refused the game. The test runs the script.

**A first visit keeps the query too.** `home.ts` adds `location.search` to
every link to `/play` (`app/home/playLinks.ts`), the Discord ones
(`/play?discord=join`, see [Discord pings](discord-alerts.md)) with their own
query kept, so a newcomer who opened a share link reaches the game with its flag
rather than with their time zone's.

**The game is a real file at each path**, not a fallback. The Workers fallback
is `index.html`, the home page, so a `/auth/callback` that relied on it would
land on the home page and lose the code. The `gameRoutes` plugin in
`vite.config.ts` writes `play.html` a second time as `auth/callback.html`, and
in `npm run dev` rewrites both paths to `play.html`. The registered redirect URI
is unchanged. Paths under `/play/` have no file and get the home page: the game
has no routes of its own.

**The bundle is named `play-*.js`** now, not `index-*.js`, and it is linked from
`/play`, not from `/`. The home page links `home-*.js` (`src/home.ts`), which
shares the season client's chunk with the game.
