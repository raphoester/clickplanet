# Every PR is a vertical slice

**A PR is one deliverable unit.** It cuts through every layer one behavior needs, from the edge to the storage,
and after its deploy someone can do or check something they could not before: a player, the operator, or a module
that calls it. A horizontal slice (a domain package, a store, a proto or a component alone, waiting for a later PR
to call it) is forbidden. It brings no value on its own, it cannot be checked in production, and code that nothing
calls fails `make deadcode` (see [Git hooks](git-hooks.md)).

- **The test before writing code:** what can someone do or check after the merge and the deploy that they could not
  before? If the answer is nothing, the work is a layer. Make it part of the slice that calls it.
- **Hidden still counts.** A slice behind a flag that is off, or one only the operator reads (an admin RPC, so a
  connector tool), runs in production and can be checked. #817 is one: the days played, from the column and the
  live count to the planet read, the backfill call and the field in the admin `GetProgress`.
- **The thinnest caller makes a layer a slice.** A new value can be read first by the operator: the admin read
  that shows it is the slice's edge.
- **A design or a request that splits the work into layers is not a reason to ship a layer.** Before writing code,
  propose the smallest vertical slice that holds it, and say why.
- **A red `make deadcode` on a PR is the sign of a horizontal slice.** Fix the slice: never merge it red, and never
  add a caller that exists only to satisfy the check.

What it cost once: #818 (2026-11-16) added `apps/backend/internal/progression/internal/levels/` alone, the XP
domain with no caller. Its `make deadcode` was red, it was merged so, and `main` stayed red until a slice called it.
