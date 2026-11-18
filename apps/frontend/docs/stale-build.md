# A page from an older build

**A deploy removes the files of the build before it.** The Workers asset handler serves the current build's `dist/`
alone, and `not_found_handling: "single-page-application"` answers any path it does not hold with `index.html`, 200,
`text/html`, under the `/assets/*` rule's year-long `immutable` cache. So a page from an older build that asks for one of
its own `/assets/` files gets HTML: a module script is refused for its type, a stylesheet is dropped, a lazy chunk's
import fails.

**What it looked like.** On 2026-11-18, a day of a dozen frontend deploys, `/play` on the owner's iPhone (Brave) showed
only the globe's sky color in a normal tab, and played in a private one. A `?diag` report later in that tab showed a
build two deploys old: `?gfx=adaptive` was ignored, as the builds before #887 ignore it. The old build's JS was in the
browser's cache; some file it needed was not, and came back as HTML. iOS restores a tab from its cache, without asking
the server, so a page kept from before a deploy runs again.

Two things keep it from happening, and from staying:

- **The pages that load `/assets/` files are never kept**: `Cache-Control: no-store` on `/`, `/index.html`, `/play`,
  `/archive`, `/video` and `/auth/callback` in `public/_headers`. Until Nov 2026 they were `max-age=0,
  must-revalidate`, which still lets a browser store the page and show it again without asking.
- **A failed `/assets/` file reloads the page once**: the second inline script in `play.html`, before every
  `/assets/` tag Vite writes into `<head>`. It hears, in the capture phase, an `error` from any element whose `src` or
  `href` is under `/assets/` (the entry script, a stylesheet, a module preload), and Vite's `vite:preloadError` (a lazy
  chunk or its CSS, which also covers a tab left open across a deploy). The reload is kept in `sessionStorage`
  (`clickplanet-stale-reload`) and not done again within 60 s, so a build that is broken itself reloads once and
  stops. With no `sessionStorage` it never reloads. When it does not reload, `vite:preloadError` is left to throw.

**Checked** on 2026-11-18 in Playwright WebKit with an iPhone profile, against production with the guard added to its
`/play`: an old entry script and stylesheet, an old stylesheet alone, and an old lazy globe chunk each reload once and
play; a page that stays old reloads once and stops; the current build never reloads.

**Not covered.** A browser that once got the HTML fallback for a file of the current build, as in a deploy's race,
keeps it a year under that name (the `immutable` above): the reload gets the same file from its cache. Only the next
deploy, which renames the file, clears it. `/static/` files read with `fetch` (the boards' files) are not watched: they
are content-addressed, and an old build asks for them only while its own files are still in the cache.
