# Arrivals (`internal/auth/internal/arrivals/`)

**Every account auth makes keeps how it came: a player's link, the YouTube link, or unknown.** The frontend is on
Cloudflare Pages, Caddy sees only API calls, Cloudflare's free analytics see no query string and TikTok sends no
referrer. So the server records the source when it makes the account, from what the browser says its first page was.

- **The browser sends its landing** (`auth.v1.Landing`): the `?f=` and the `?s=` of the first page of the game it
  opened, kept in local storage from then on (see the frontend's [Arrivals](../../frontend/docs/arrivals.md)). It
  goes with every call that can make an account: `CreateSession`, `CompleteSignIn` and `CompleteEmailSignIn`.
- **Only a call that makes an account keeps it.** `arrivals.Guests` starts the guest and records its arrival in one
  transaction, for `CreateSession`; `signin.Admitter` records it in the sign-in's transaction when the outcome is
  `Created`. A resumed session, a sign-in to an account that exists and a link keep nothing: the account's arrival
  was kept when it was made.
- **`arrivals.NewArrival` decides the source**, from the landing alone. `?s=yt` is `youtube`; else a flag that is a
  country (`cpcountries`) is `recruit`; else `unknown`. The flag is kept for every source, `""` when the link named
  none, so a YouTube link that names a flag keeps it too. An unknown tag is ignored, an unknown flag is `""`.
- **What counts as a recruit**: the first page of the game had `?f=<flag>` and no `?s=`. No recruiter is kept.
  Every `?f=` link the game builds is a player's: `shareUrl` in `src/domain/shareCard.ts`, shown by `RecruitLink`
  and drawn into the shared picture (`drawShareCard`, `shareText`). The home page carries the query to `/play`.
- **TikTok has no tag**: its bio link cannot be clicked under 1,000 followers, so people type the address, and a
  query would cost conversion. Those arrivals are `unknown`.
- **The YouTube link** is `https://clickplanet.lol/?s=yt`: the channel page's link, set by hand in YouTube Studio,
  and the description of every clip the clips job posts (`youtubeSite`, see [The clips job](clips-job.md)).
- **One table, `auth.arrivals`**: `id`, `account_id` (`NULL` once anonymized), `source` (the enum `arrival_source`:
  `recruit`, `youtube`, `unknown`), `flag`, `arrived_at`. Migration `20261030120000_arrivals`. Accounts made before
  it have no row.
- **An arrival outlives its account.** It is its own concept with its own store, not a column of `accounts`: auth
  hears its own `auth.v1.AccountDeleted` (consumer group `auth-arrivals-accounts`), and
  `anonymize_arrivals_usecase` blanks the account (`Arrival.Anonymized`) and keeps the source, the flag and the
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

Add `r.flag` to the inner select and to `GROUP BY` to split the recruits by flag. The `EXISTS` reads one account's
ledger rows through `ledger_events_account_kind`, so it stays cheap at 25 new accounts a day.

No backoffice screen reads it yet. When one does, it gets the raw arrivals and counts them itself (see
[Reads are queries](queries.md)).
