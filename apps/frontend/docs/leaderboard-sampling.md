# Sampling the leaderboard

The globe re-ranks on every count it takes in from the server — the read at load,
and `holdings_changed` at most once a second — and `useLeaderboardFeed` is what
stops each of those reaching React at once. Rendering two hundred rows, and restarting two hundred badge
animations, ten times a second is enough to make the whole machine stutter on a
busy planet, and it is unreadable anyway: a count nobody can follow flickering
past.

**The accounting still follows every batch; only the render is sampled.**
`recordLeaderboard` folds each board into the badges in a ref — pure arithmetic
over a couple of hundred numbers, no DOM — and a `SAMPLE_MS` interval publishes
what has accumulated. That split is what makes the badges both cheap and honest:

- A country that won three tiles in half a second says **"+3" once**, rather
  than "+1" three times too fast to read.
- The initial load is told apart from live play **exactly**, which sampling the
  stream alone could not do. `createGlobe` passes `live: false` for a read of the
  counts (at load, or after a gap), and `live: true` for each change on the
  stream. Folding per board keeps a read out of the badges while still counting
  it into the ground they are measured from.
- A tick with nothing to publish and nothing to retire **calls no setter at
  all** — handing React the state it already holds still costs a render before
  it bails out.

The cost is up to a second and a half of staleness on the tiles count, including
your own click, which counts once the server has. The tile itself paints
immediately, which is the feedback that matters; the leaderboard is not read that
fast.
