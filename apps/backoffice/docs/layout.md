# Layout

```
src/
  api.ts            the transport, one client per AdminService, LONG_CALL_MS, putFile (a file to a signed bucket link)
  clients.ts        the narrow client types every page takes, and CLIENTS
  fakeAdmin.ts      FakeAdmin, every admin method for the tests
  demo/             npm run demo: api.ts, the clients over DemoAdmin (demoAdmin.ts) and a made-up bucket, served in
                    place of ../api.ts
  refusal.ts        a page's error → signedOut, notOperator, startAgain or failed; an action's in words (wordsOf)
  useCall.ts        one call's state, run on mount
  useAction.ts      one read's state, run on a click
  useRoute.ts       the page shown and what is open on it, kept in the address bar
  useWide.ts        whether the window is wide enough for the sidebar
  usePageVisible.ts whether the tab is shown
  navigation.ts     MENU, the pages by group, the route (routeOf, hashOf) and OpenPlayer
  countries.ts      a country id's name and its flag
  App.tsx           the callback, or WhoAmI first, then the page the menu picked
  signIn/           callbackOf, SignInButton, SignInCallback, SignOutButton
  components/       Menu and Sidebar, ConfirmAction, FailureNotice, AccountId and Copyable (with Copy), Country, CountryField,
                    Fact, Person, When and Ago, Fold (a panel read once opened), ChangeList (what a form changes)
  home/             Home: the tiles, ActivityCharts, Heatmap, CountryTable, PlayingNow, over ledger.ts (the counts),
                    heat.ts (the picture), places.ts (the tile coordinates) and activity.ts
  support/          Ideas and bugs: NoteList, Attachments, labels.ts (names, outcomes, device lines, sizes)
  chat/             Conversation and TheirMessages, which reports/ and players/ show, over chatPages.ts (a page of
                    ListMessages, the parties a report names) and theirConversations.ts
  reports/          Reports: ReportsView (the inbox), ReportList, ReportDetail, VerdictPanel over verdict.ts, actions.ts
  players/          Players: list, PlayerName, PlayerView, Standing, ProfilePanel, the tools on one player, actions.ts
  map/              Reassign a country, Paint random tiles, Download a replay
  clips/            Clip stats: ClipStatsPage, actions.ts (the CSV's file), table.ts (its columns)
  videos/           Videos: VideosPage (the list), UploadPanel, actions.ts (the upload's steps, copyText)
  ops/              Reconcile titles, Game rules, Anti-bot rules (a form read off the proto), Rule schedule, Seasons,
                    Timeline
  gen/grpc/         generated from ../../proto, committed
  gen/map/          the tile coordinates, copied from ../../map, committed
```

**An account is shown by its id, with Copy**, beside its name, wherever a tool takes it: a name can change hands,
and the id is what the operator tools take. A report shows each id short (its first group, the whole on hover), and
Copy copies the whole. **A list is the exception**: its rows show the flag and the name alone, and the id is on the
page each opens.

Plain CSS in `src/index.css`, tokens on `:root`, dark from `prefers-color-scheme`. The page is up to 1280px wide, so a
list and the one open sit side by side on a wide screen. **It must work on a phone**: 16px gutter, and nothing may
scroll sideways, so every id and text wraps (`overflow-wrap: anywhere`), and a table hides its wide columns
(`col-wide`) below 720px.
