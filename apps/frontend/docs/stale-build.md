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
- **A failed `/assets/` file reloads the page, at most twice in a minute**: the second inline script in `play.html`,
  before every `/assets/` tag Vite writes into `<head>`. It hears, in the capture phase, an `error` from any element
  whose `src` or `href` is under `/assets/` (the entry script, a stylesheet, a module preload), and Vite's
  `vite:preloadError` (a lazy chunk or its CSS, which also covers a tab left open across a deploy). One page reloads
  once, however many of them fire.
  - **The first reload** is a plain one: the page is never kept, so it comes back current, and an older build's
    files are not asked for again.
  - **The second, within 60 s of the first**, asks again for every `/assets/` file the page asked for (its tags and
    the Resource Timing entries), with `cache: "reload"`, then reloads, after them or after 3 s. This is for a
    browser that once got the HTML fallback for a file of the current build, as in a deploy's race, and keeps it a
    year under that name (the `immutable` above): the request past the cache replaces it with the file itself. Every
    file is asked for because WebKit's error for a lazy chunk does not name it.
  - **Then it stops**: a build that is broken itself reloads twice, not forever. The tries are kept in
    `sessionStorage` (`clickplanet-stale-reload`, `{at, tries}`); with no `sessionStorage` it never reloads. When it
    does not reload, `vite:preloadError` is left to throw.

**Why not a 404 for a missing `/assets/` file.** The fallback cannot be turned off: a tracked address
(`/t`, `/r?f=…`) or a player's old one (`/<flag>/<name>`) has no file and reaches the game through it (see [Pages and routes](routes.md)). A 404 for
`/assets/` alone takes a Worker script run before those paths, and on Workers' free plan a day past its request limit
fails every request that runs it: every file of the page. A 404 would also carry the `/assets/*` rule's `immutable`.

**Checked** on 2026-11-18 in Playwright WebKit with an iPhone profile, against production with the guard swapped into
its `/play`:

| Case | Loads of `/play` | Then |
|---|---|---|
| an old entry script and stylesheet | 2 | plays |
| the current globe chunk answered with the fallback once, kept `immutable` | 3 | plays, the globe drawn |
| the current entry script, or stylesheet, answered with the fallback once | 2 | plays |
| a page that stays old | 3 | stops |
| the current build | 1 | plays |

**Not covered.** `/static/` files read with `fetch` (the boards' files) are not watched: they are content-addressed,
and an old build asks for them only while its own files are still in the cache.
