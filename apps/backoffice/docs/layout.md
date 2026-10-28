# Layout

```
src/
  api.ts            the transport, one client per AdminService, LONG_CALL_MS
  clients.ts        the narrow client types every page takes, and CLIENTS
  fakeAdmin.ts      FakeAdmin, every admin method for the tests
  refusal.ts        a page's error → signedOut, notOperator, startAgain or failed; an action's in words (wordsOf)
  useCall.ts        one call's state, run on mount
  useAction.ts      one read's state, run on a click
  usePage.ts        the page shown, kept in the address bar
  useWide.ts        whether the window is wide enough for the sidebar
  usePageVisible.ts whether the tab is shown
  navigation.ts     MENU, the pages by group, and OpenPlayer
  countries.ts      a country id's name and its flag
  App.tsx           the callback, or WhoAmI first, then the page the menu picked
  signIn/           callbackOf, SignInButton, SignInCallback, SignOutButton
  components/       Menu and Sidebar, ConfirmAction, FailureNotice, AccountId and Copyable (with Copy), Country, CountryField,
                    Fact, Person, When
  home/             Home: the tiles, ActivityCharts, Heatmap, CountryTable, PlayingNow, over ledger.ts (the counts),
                    heat.ts (the picture), places.ts (the tile coordinates) and activity.ts
  support/          Ideas and bugs: NoteList, Attachments, labels.ts (names, outcomes, device lines, sizes)
  chat/             Conversation and TheirMessages, which reports/ and players/ show, over chatPages.ts (a page of
                    ListMessages, the parties a report names) and theirConversations.ts
  reports/          Reports: list, detail, the report's own actions
  players/          Players: list, PlayerName, PlayerView, ProfilePanel, the tools on one player, actions.ts
  map/              Reassign a country, Paint random tiles, Download a replay
  ops/              Reconcile titles, Game rules, Anti-bot rules, Rule schedule, Seasons, Timeline
  gen/grpc/         generated from ../../proto, committed
  gen/map/          the tile coordinates, copied from ../../map, committed
```

**An account is shown by its id, with Copy**, beside its name, wherever a tool takes it: a name can change hands,
and the id is what the operator tools take. **A list of players is the exception**: its rows show the flag and the
name alone, and the id is on the player page each opens.

Plain CSS in `src/index.css`, tokens on `:root`, dark from `prefers-color-scheme`. **It must work on a phone**:
16px gutter, and nothing may scroll sideways, so every id and text wraps (`overflow-wrap: anywhere`).
