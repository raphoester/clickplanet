# Arrivals (`internal/auth/internal/arrivals/`)

**Every account auth makes keeps how it came: a player's link, the YouTube link, CrazyGames, or unknown, the host the
browser came from, and, for a player's address with a name, the account of the player who sent it.** The frontend is a Cloudflare Worker, Caddy sees only API calls, Cloudflare's free analytics see
no query string and TikTok sends no referrer. So the server records it when it makes the account, from what the
browser says its first page was.

- **The browser sends its landing** (`auth.v1.Landing`): the `?s=` of the first page of the site it opened, the
  host of the page before it, and `recruiter_account_id` (proto `optional`) when that page was a player's address with
  a name, kept in local storage from then on (see the frontend's [Arrivals](../../frontend/docs/arrivals.md)). It goes with every call that can make an account: `CreateSession`,
  `CompleteSignIn`, `CompleteEmailSignIn` and `CompleteCrazyGamesSignIn`.
- **Only a call that makes an account keeps it.** `arrivals.Guests` starts the guest and records its arrival in one
  transaction, for `CreateSession`; `signin.Admitter` records it in the sign-in's transaction when the outcome is
  `Created`. A resumed session, a sign-in to an account that exists and a link keep nothing: the account's arrival
  was kept when it was made.
- **`arrivals.NewArrival` decides the source from the tag alone**: `rc` is `recruit`, `yt` is `youtube`, `cg` is
  `crazygames`, anything else is `unknown`, since anybody can type a tag. The CrazyGames page sends `cg` on every
  call, whatever its link said. **Auth knows no flag**: `?f=` is the game's, and the client keeps
  it to itself.
- **The tag itself is kept too** (`tagOf`), lower case, for every source: a link made for one streamer
  (`?f=pl&s=2poko2`, since 2026-11-11) is `unknown` and its tag names the stream. Only a short plain word is kept:
  32 at most, letters, digits, `_` and `-`; anything else is `""`, so a typed tag cannot carry a sentence or a script.
- **What counts as a recruit**: the first page was a flag's address, `clickplanet.lol/<flag>` or
  `clickplanet.lol/<flag>/<name>`, whose first script opens the game with `?s=rc` (see the frontend's [Pages and
  routes](../../frontend/docs/routes.md)), or an old link with `?s=rc`. Every share a player makes is such an address
  (`shareUrl` in `src/domain/shareCard.ts`): the recruit link, the frozen map's Invite, the text that rides with a
  picture and the words drawn into it, so a link typed from a picture is a recruit too. An old `?f=` link with no tag
  is `unknown`, as before.
- **The recruiter is an account, never a name.** Auth asks no module, and player depends on auth, so the browser
  reads the name in the address into the account that holds it with player's public `GetPlayerByName` (see
  [Player](player.md)), and sends that account in the landing. `landingmessage.LandingOf` reads it: an id that is not
  an account is no recruiter and fails no call, since anybody can type an address. A guest's address has no name, and
  keeps no recruiter, nor does a name nobody holds.
- **Only a recruit has a recruiter** (`Arrival.Recruitment`): a landing of another source that names one keeps none.
  **It has a table of its own, `auth.recruitments`** (`arrival_id`, the arrival's id as its key, and `recruiter_id`,
  `NULL` once anonymized; migration `20261112110331_recruitments`), so no other arrival has a recruiter column to leave
  empty. `arrivals.Register` saves it after the arrival, in the same transaction, through its own store
  (`arrivals.Recruitments`, `postgres_recruitment_store`). The migration was `20261111120000_recruitments`: it merged
  after `20261111140000_arrival_tag` had deployed, so production skipped it. The table was made by hand there on
  2026-11-12, then the migration moved to a later version, idempotent.
- **The referrer is a host and nothing else** (`hostOf`): lower case, letters, digits, dots and hyphens, 253 at
  most; anything else, a path or a port included, is `""`. The browser sends only the host, and the server checks
  again. It is kept for every source, `""` when the browser sent none: an in-app browser, a typed address, and the
  game's own pages.
- **TikTok has no tag**: its bio link cannot be clicked under 1,000 followers, so people type the address, and a
  query would cost conversion. Those arrivals are `unknown`.
- **The YouTube link** is `https://clickplanet.lol/?s=yt`: the channel page's link, set by hand in YouTube Studio,
  and the description of every clip the clips job posts (`youtubeSite`, see [The clips job](clips-job.md)).
- **One table, `auth.arrivals`**: `id`, `account_id` (`NULL` once anonymized), `source` (the enum `arrival_source`:
  `recruit`, `youtube`, `unknown`, and `crazygames` since `20261031120000_crazygames`), `tag` (since
  `20261111140000_arrival_tag`; `''` on the rows made before it), `referrer`, `arrived_at`.
  Migration `20261030120000_arrivals`. Accounts made before it have no row.
- **An arrival outlives its account.** It is its own concept with its own store, not a column of `accounts`: auth
  hears its own `auth.v1.AccountDeleted` (consumer group `auth-arrivals-accounts`), and
  `anonymize_arrivals_usecase` blanks the account (`Arrival.Anonymized`) and keeps the source, the tag, the referrer
  and the time. A deleted account and a guest pruned after 90 days idle go the same way, so a day's count never shrinks.
- **A recruitment outlives both accounts.** The recruit's account goes with its arrival's: the recruitment names it
  only through the arrival, which names nobody once anonymized. The recruiter's goes on its own group,
  `auth-recruitments-accounts`: `anonymize_recruitments_usecase` blanks the recruiter of each recruitment it made
  (`Recruitment.Anonymized`) and keeps the row, so the arrival still counts as one a player brought.
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

Add `r.referrer` to the inner select and to `GROUP BY` to split each source by the host it came from, or `r.tag` to
split it by link: one row per streamer's link. The `EXISTS` reads one account's ledger rows through
`ledger_events_account_kind`, so it stays cheap at 25 new accounts a day.

Who brought the most new accounts in the last 14 days:

```sql
SELECT p.recruiter_id, count(*) AS recruits
FROM auth.recruitments p
JOIN auth.arrivals r ON r.id = p.arrival_id
WHERE p.recruiter_id IS NOT NULL AND r.arrived_at > now() - interval '14 days'
GROUP BY p.recruiter_id
ORDER BY recruits DESC;
```

A recruit with no row in `auth.recruitments` came by a flag's address with no name, or by an old link.

No backoffice screen reads it yet. When one does, it gets the raw arrivals and counts them itself (see
[Reads are queries](queries.md)).
