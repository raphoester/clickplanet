# Sampling the leaderboard

The globe re-ranks on every batch it takes in — ten times a second, over every
country on the map — and `useLeaderboardFeed` is what stops each of those
reaching React. Rendering two hundred rows, and restarting two hundred badge
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
  stream alone could not do. `createGlobe` passes `live: false` for the ~26
  batches of the initial fetch, and live updates stream in throughout, so a
  half-second window routinely holds both. Folding per batch keeps the map as it
  loads out of the badges while still counting it into the ground they are
  measured from.
- A tick with nothing to publish and nothing to retire **calls no setter at
  all** — handing React the state it already holds still costs a render before
  it bails out.

The cost is up to half a second of staleness on the tiles count, including your
own click. The tile itself paints immediately, which is the feedback that
matters; the leaderboard is not read that fast.
