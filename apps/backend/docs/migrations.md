# Migrations

**Each module keeps its migrations in `internal/<module>/internal/migrations`**, embedded golang-migrate pairs named
`<yyyymmddhhmmss>_<snake_case>.up.sql` and `.down.sql`, with the UTC time it was written as the version. The module
runs them at boot, in its own schema (see [Durability](durability.md)). A new migration takes the current UTC time,
never a round guess like `120000`: two sessions on the same day pick the same one.

## What golang-migrate does with them

**It keeps one number per schema: the newest version it ran** (`schema_migrations`, one row). At boot it runs every
version after that number, in order, and nothing before it. So:

- **A migration older than the newest that ran is skipped, with no error.** This happens when two branches each add a
  migration, and the one with the later version deploys first. On 2026-11-12, #772 (`20261111140000_arrival_tag`)
  deployed at 10:08 UTC, and #766 (`20261111120000_recruitments`) at 10:46 UTC: production never made
  `auth.recruitments`.
- **The version the database is at must still be in the code**, or the boot fails (`no migration found for version`).
- **Two files of one version and direction fail the boot** (`duplicate migration file`).
- **A name it cannot read is skipped, with no error.** A version with 13 digits reads as a much older one.

## The check

`make check-migrations` (`cmd/migrationcheck`) reads each module's migrations from the working tree and refuses:

- a name that is not `<yyyymmddhhmmss>_<snake_case>.up.sql` or `.down.sql`, or a version that is not a date and time;
- two migrations (two names) on one version;
- a version with no `.up.sql`.

With `BASE=<git revision>`, it takes a database that ran every migration of that revision, and also refuses:

- a migration added since that is older than the newest one in the revision: golang-migrate would skip it;
- the newest one in the revision, gone: the boot would fail.

A module that is new, or gone (a rename moves the directory), is not compared.

**Where it runs**, each time with the base that fits:

- **pre-push**: `BASE=origin/main`, as last fetched.
- **`check backend` on a pull request**: `BASE=HEAD^1`, main as it is when the job runs. Stricter than the deploy:
  main's newest migration may not have deployed yet. On a push to main, only the names and versions.
- **`deploy backend`, before the images build**: the `migrations` job reads the revision the VPS runs, from the label
  `org.opencontainers.image.revision` the build puts on the image (`docker inspect cp-backend` over ssh), and checks
  against it. This is the one that counts: a pull request checked green can still merge after a newer migration
  deployed. When it fails, nothing is built or rolled. An image built before 2026-11-12 has no label: the job then
  checks only the names and versions, with a warning.

**When it fails**, give the new migration the current UTC time as its version. Never rename a migration that ran in
production: the database keeps its number.

**It cannot see a skip that already happened.** It trusts that the database ran every migration of the base. A
migration skipped before the check existed stays skipped: move it to a new version, with `IF NOT EXISTS` where a
database may already hold what it makes.
