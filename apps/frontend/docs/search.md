# Search

About 93% of new players come with no link: they type the address or search the name after a clip. So Google must
know the name in every spelling, say plainly what the game is, and show one site: clickplanet.lol.

## What Google reads

- **Titles.** `/` is `ClickPlanet · Paint the planet with your flag`, `/play` is
  `Play ClickPlanet · The live flag war on a world map`, `/archive` is `ClickPlanet archive`.
- **One description** in `meta name=description`, `og:description` and `twitter:description` of `index.html`,
  `play.html` and `archive.html`, and in the JSON-LD's `VideoGame`: what the game is, and why it is not an idle
  clicker (land taken from other countries, on one live map). Change all of them together. Plain words: a poetic line
  reads as AI.
- **Structured data**: one JSON-LD block in the head of `index.html`.
  - `WebSite` with `alternateName` "Click Planet" and "clickplanet.lol", as Google's
    [site names](https://developers.google.com/search/docs/appearance/site-names) guide asks. This tells Google
    that "click planet" in two words is our name.
  - `VideoGame`: name, url, description, the preview image, genre, `gamePlatform`, `playMode`, free `offers`.
  - **Only true facts.** No rating, no review count, no player count. Google's software-app result asks for a rating,
    so we get no stars. Never add a rating we did not collect.
  - `npm run og-image` rewrites every `/static/og-image-*.jpg` in the three pages, so the JSON-LD image follows it.
- **Canonicals.** Each page names its own. A flag's address (`/pl/kutza`) is the home page, canonical `/`.
- **Sitemap** (`public/sitemap.xml`): `/`, `/play`, `/archive`, `/privacy`, `/terms`. Add a page when it is a real
  public page with its own canonical. Submitted in Search Console on Nov 13 2026.
- **robots.txt** allows everything and names the sitemap (see [Static assets](static-assets.md)).

## The CrazyGames host is out of Google

`crazygames.clickplanet.lol` is the same build (see [CrazyGames](crazygames.md)). Before Nov 2026 its `/` and
`/terms` showed in Google (92 impressions) and took clicks from the main site.

`public/_headers` sends `X-Robots-Tag: noindex` on `https://crazygames.clickplanet.lol/*`. Workers static assets
accept a host in a `_headers` pattern ([Cloudflare](https://developers.cloudflare.com/workers/static-assets/headers/)).
The project has no Worker script, so every answer is a static asset and gets the header, the home page fallback too.
The header does not change framing or caching, so the CrazyGames frame works as before. **Never send noindex on
clickplanet.lol.**

## Baseline (Search Console, Sep 16 to Nov 10 2026)

| Search | Position | Click-through |
|---|---|---|
| "clickplanet" | 1.19 | 86% |
| "clickplanet lol" | 1.02 | 97% |
| "click planet" | 2.68 | 56% |
| "click the planet", "clicksplanet", "planet click", "click a planet" | 5 to 8 | 0 clicks |
| On a computer | 3.06 | 41% |
| On a phone | 1.51 | 83% |

"click planet" lost about 300 searches in 3 months: Google filled that page with the idle game "Planet Clicker".
Index coverage had no errors.

**Goal**, read 2 to 4 weeks after the Nov 2026 change: "click planet" 56% → 75% or more, computers 41% → 60% or more,
no CrazyGames page in the results. About 10 to 20 more visits a day.

## Reading Search Console

With the Composio CLI, on the owner's Google account. The site is `sc-domain:clickplanet.lol`. Use only these read
slugs; submitting or removing anything is the owner's call. Data comes 2 to 3 days late.

```bash
composio execute GOOGLE_SEARCH_CONSOLE_SEARCH_ANALYTICS_QUERY -d '{"site_url": "sc-domain:clickplanet.lol", "start_date": "2026-11-14", "end_date": "2026-12-12", "dimensions": ["query"], "row_limit": 100}'
```

`"dimensions": ["device"]` splits computers from phones, `["page"]` shows which host and page Google sends people to.

```bash
composio execute GOOGLE_SEARCH_CONSOLE_GET_SITEMAP -d '{"site_url": "sc-domain:clickplanet.lol", "feedpath": "https://clickplanet.lol/sitemap.xml"}'
```

```bash
composio execute GOOGLE_SEARCH_CONSOLE_INSPECT_URL -d '{"site_url": "sc-domain:clickplanet.lol", "inspection_url": "https://clickplanet.lol/"}'
```
