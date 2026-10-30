# Arrivals (`internal/auth/internal/arrivals/`)

**Every account auth makes keeps how it came: a player's link, the YouTube link, CrazyGames, or unknown, and the host
the browser came from.** The frontend is on Cloudflare Pages, Caddy sees only API calls, Cloudflare's free analytics see
no query string and TikTok sends no referrer. So the server records it when it makes the account, from what the
browser says its first page was.

- **The browser sends its landing** (`auth.v1.Landing`): the `?s=` of the first page of the site it opened and the
  host of the page before it, kept in local storage from then on (see the frontend's
  [Arrivals](../../frontend/docs/arrivals.md)). It goes with every call that can make an account: `CreateSession`,
  `CompleteSignIn`, `CompleteEmailSignIn` and `CompleteCrazyGamesSignIn`.
- **Only a call that makes an account keeps it.** `arrivals.Guests` starts the guest and records its arrival in one
  transaction, for `CreateSession`; `signin.Admitter` records it in the sign-in's transaction when the outcome is
  `Created`. A resumed session, a sign-in to an account that exists and a link keep nothing: the account's arrival
  was kept when it was made.
- **`arrivals.NewArrival` decides the source from the tag alone**: `rc` is `recruit`, `yt` is `youtube`, `cg` is
  `crazygames`, anything else is `unknown`, since anybody can type a tag. The CrazyGames page sends `cg` on every
  call, whatever its link said. **Auth knows no flag**: `?f=` is the game's, and the client keeps
  it to itself.
- **What counts as a recruit**: the first page had `?s=rc`. No recruiter is kept. `shareUrl` in
  `src/domain/shareCard.ts` adds it to every link a player sends: the recruit link (`RecruitLink`), the frozen map's
  Invite (`RecruitButton`) and the text that rides with a picture (`shareText`). **The picture's own words have no
  tag** (`shareLabel`): a longer one meets the logo on a portrait card, so a link typed from a picture is `unknown`.
  The home page carries the query to `/play`.
- **The referrer is a host and nothing else** (`hostOf`): lower case, letters, digits, dots and hyphens, 253 at
  most; anything else, a path or a port included, is `""`. The browser sends only the host, and the server checks
  again. It is kept for every source, `""` when the browser sent none: an in-app browser, a typed address, and the
  game's own pages.
- **TikTok has no tag**: its bio link cannot be clicked under 1,000 followers, so people type the address, and a
  query would cost conversion. Those arrivals are `unknown`.
- **The YouTube link** is `https://clickplanet.lol/?s=yt`: the channel page's link, set by hand in YouTube Studio,
  and the description of every clip the clips job posts (`youtubeSite`, see [The clips job](clips-job.md)).
- **One table, `auth.arrivals`**: `id`, `account_id` (`NULL` once anonymized), `source` (the enum `arrival_source`:
  `recruit`, `youtube`, `unknown`, and `crazygames` since `20261031120000_crazygames`), `referrer`, `arrived_at`.
  Migration `20261030120000_arrivals`. Accounts made before it have no row.
- **An arrival outlives its account.** It is its own concept with its own store, not a column of `accounts`: auth
  hears its own `auth.v1.AccountDeleted` (consumer group `auth-arrivals-accounts`), and
  `anonymize_arrivals_usecase` blanks the account (`Arrival.Anonymized`) and keeps the source, the referrer and the
  time. A deleted account and a guest pruned after 90 days idle go the same way, so a day's count never shrinks.
- **What it gets wrong**: a second account made in the same browser (after a sign-out, or a guest whose cookie
  lapsed) gets the first page's landing too. A browser that first opened the game before this shipped keeps the
  landing of its first page after it.

## Counting them

New accounts per day (UTC) by source, and how many took a tile again the next day. `back_next_day` is empty until
that next day is over. An anonymized arrival never counts as back: the ledger lost its account too. Run on the
box with `docker exec -i cp-postgres psql -U clickplanet -d clickplanet`:

```sql
SELECT a.day,
       a.source,
       count(*) AS new_accounts,
       CASE WHEN a.day + 1 < (now() AT TIME ZONE 'UTC')::date
            THEN count(*) FILTER (WHERE a.back) END AS back_next_day,
       CASE WHEN a.day + 1 < (now() AT TIME ZONE 'UTC')::date
            THEN round(100.0 * count(*) FILTER (WHERE a.back) / count(*)) END AS back_pct
FROM (
    SELECT (r.arrived_at AT TIME ZONE 'UTC')::date AS day,
           r.source,
           EXISTS (
               SELECT 1 FROM planet.ledger_events e
               WHERE e.account = r.account_id
                 AND e.taken_at >= ((r.arrived_at AT TIME ZONE 'UTC')::date + 1)::timestamp AT TIME ZONE 'UTC'
                 AND e.taken_at <  ((r.arrived_at AT TIME ZONE 'UTC')::date + 2)::timestamp AT TIME ZONE 'UTC'
           ) AS back
    FROM auth.arrivals r
) a
GROUP BY a.day, a.source
ORDER BY a.day, a.source;
```

Add `r.referrer` to the inner select and to `GROUP BY` to split each source by the host it came from. The `EXISTS` reads one account's
ledger rows through `ledger_events_account_kind`, so it stays cheap at 25 new accounts a day.

No backoffice screen reads it yet. When one does, it gets the raw arrivals and counts them itself (see
[Reads are queries](queries.md)).
