# Layout

```
src/
  api.ts            the transport, one client per AdminService, LONG_CALL_MS, putFile (a file to a signed bucket link)
  clients.ts        the narrow client types every page takes, and CLIENTS
  fakeAdmin.ts      FakeAdmin, every admin method for the tests
  demo/             npm run demo: api.ts, the clients over DemoAdmin (demoAdmin.ts) and a made-up bucket, served in
                    place of ../api.ts
  refusal.ts        a page's error → signedOut, notOperator, startAgain or failed; an action's in words (wordsOf),
                    the server's UTC times in the operator's own (localized)
  useCall.ts        one call's state, run on mount
  useAction.ts      one read's state, run on a click
  useRoute.ts       the page shown and what is open on it, kept in the address bar
  useWide.ts        whether the window is wide enough for the sidebar
  usePageVisible.ts whether the tab is shown
  navigation.ts     MENU, the pages by group, the route (routeOf, hashOf) and OpenPlayer
  countries.ts      a country id's name and its flag
  main.tsx          takes the sign-in callback and a connector's request off the address bar, then renders App
  App.tsx           the callback, or WhoAmI first, then the page the menu picked
  signIn/           callbackOf, SignInButton, SignInCallback, SignOutButton
  connect/          ConnectPage (Allow or Deny an app), pending.ts (the request kept for the tab)
  components/       Menu and Sidebar, ConfirmAction, FailureNotice, AccountId and Copyable (with Copy), Country, CountryField,
                    Fact, Person, When and Ago, Fold (a panel read once opened), ChangeList (what a form changes);
                    time.ts (a time in words, a span of two, as a datetime-local holds it, the operator's time zone)
  home/             Home: the tiles, ActivityCharts, Heatmap, CountryTable, PlayingNow, over ledger.ts (the counts),
                    heat.ts (the picture), places.ts (the tile coordinates) and activity.ts; useWidth.ts (a chart's
                    width as it resizes)
  support/          Ideas and bugs: NoteList, Attachments, labels.ts (names, outcomes, device lines, sizes)
  chat/             Conversation and TheirMessages, which reports/ and players/ show, over chatPages.ts (a page of
                    ListMessages, the parties a report names) and theirConversations.ts; said.ts (what a message said,
                    in words)
  reports/          Reports: ReportsView (the inbox), ReportList, ReportDetail, DecisionLine (a report's outcome, time and
                    comment), VerdictPanel over verdict.ts, actions.ts
  players/          Players: list, PlayerName, PlayerView, Standing, ProfilePanel, the tools on one player, actions.ts,
                    HistoryPanel over history.ts (an account's moderation history: the chat's and the renames);
                    caller.ts (an account or a scope, from what was typed), charges.ts (a grant and what is held, in
                    words), durations.ts (the mute and ban choices, a span in words)
  map/              Reassign a country, Paint random tiles, Download a replay; download.ts (a file saved from memory)
  clips/            Clip stats: ClipStatsPage, actions.ts (the CSV's file), table.ts (its columns)
  videos/           Videos: VideosPage (the list), UploadPanel, actions.ts (the upload's steps, copyText)
  promo/            Promo, the battle teasers: PromoPage, BackgroundsPanel (the library and its uploads), CardsPanel (a set's cards,
                    edited and previewed), actions.ts (the upload's steps, the refusals, each set's values, a card's
                    problems), fill.ts (a card's words filled as the server fills them)
  ops/              Reconcile titles, Link Discord members (discordMembers.ts, its words), Game rules, Box & quiz rules,
                    Anti-bot rules (a form read off the proto), Mission rules (missionRules.ts, the four numbers), Rule
                    schedule, Seasons, Timeline; cutoff.ts (the daily cutoff: when days end, a battle off it); rules.ts
                    (a planet set in words and as the form edits it), bonusRules.ts (the offers' set, the same way),
                    RuleFields.tsx (the two forms' switch and value fields) over ruleValues.ts (a value as typed, sent
                    and read), antibotRules.ts (the antibot form's fields, read off the proto), overlay.ts (a set's
                    JSON over default's), schedule.ts (the schedule's rows and what each module plays), seasons.ts (the
                    seasons' rows and form), timeline.ts (the Timeline's lanes and instants)
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
(`col-wide`) below 720px, or turns each row into a card (the rule schedule, `.schedule`).
