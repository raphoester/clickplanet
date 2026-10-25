# Pages and routes

Two pages, built by Vite as a multi-page app (`build.rollupOptions.input` in
`vite.config.ts`):

| Path | File | What |
|---|---|---|
| `/` | `index.html` | The home page. Plain HTML, and one small module for the Final Battle. |
| `/play` | `play.html` | The game. |
| `/auth/callback` | `auth/callback.html` | The game again, for the sign-in callback. |
| `/privacy`, `/terms` | `privacy.html`, `terms.html` | Plain pages, no bundle. |
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
a share link such as `/?f=de` still reaches the game with its query. Crawlers
have no storage and read the home page. `homePage.test.ts` pins the key in the
script to the constant. **`/#home` does not redirect**: it is the "Home page"
link at the bottom of the About modal.

**A first visit keeps the query too.** `home.ts` copies `location.search` onto
every Play link (`app/home/playLinks.ts`), so a newcomer who opened a share link
reaches the game with its flag rather than with their time zone's.

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
