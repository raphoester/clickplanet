# Every action, on every page

- **Every action that changes something asks first** (`components/ConfirmAction`): the button opens a question
  with "Yes, …" and Cancel, and only Yes calls the server. Then it says what the server answered ("Muted until
  …"), or the refusal. **Steps the operator picks together ask one question that names each** (a report's verdict:
  "Remove the message, mute Ana for 1 day and close this report as Acted?"), run in order, and stop at the first
  refused, saying what was done before it. **A read asks nothing** (inspect, a dry run, a list, a replay): `useAction` runs it on a
  click and holds its answer, as `useCall` does on mount.
- **What can be undone by nothing counts first**: a revert, a reassign and a paint stay disabled until their dry
  run, and the question quotes it ("Give back the 120 tiles Ana holds?"). The dry run holds for the inputs it ran
  on alone: changing a country disables the action again, and **a question whose inputs change under it is taken
  back** (`ConfirmAction` with `disabled`).
- **The refusals in plain words** (`wordsOf` in `refusal.ts`, and each section's `refusalOf` over it, which says
  what its `not_found` and `failed_precondition` mean): `not_found` is a message or a report no longer kept, or no
  ban left to lift; `failed_precondition` is the antibot off (no ban, unban or inspection); `invalid_argument`
  quotes the server (an unknown country, a grant of nothing); `deadline_exceeded` says the call may have done part
  of its work; a session that ended says to sign in again.
- **The long calls take `LONG_CALL_MS`** (`api.ts`, 2 minutes) instead of the transport's 10s: a revert, a
  reassign and a paint are paced on the server, a replay is MBs, and the two ops runs page through every player.
- **Every page takes its clients** (`clients.ts`: `AntibotAdmin`, `ChatAdmin`, `PlanetAdmin`, `PlayerAdmin`, `GamemasterAdmin`, `SupportAdmin`,
  defaulting to `CLIENTS`), so a test hands it `FakeAdmin` (`fakeAdmin.ts`), which answers every method, records
  each request (`askedFor`), can refuse a method with a code, and can hold its answers until released (`hold`). The jsdom tests drive each action through its
  page.
