# Layout

```
src/
  api.ts            the transport, one client per AdminService, LONG_CALL_MS, putFile (a file to a signed bucket link)
  clients.ts        the narrow client types every page takes, and CLIENTS
  fakeAdmin.ts      FakeAdmin, every admin method for the tests
  demo/             npm run demo: api.ts, the clients over DemoAdmin (demoAdmin.ts) and a made-up bucket, served in
                    place of ../api.ts; demoLab.ts (ten made-up days of the lab)
  refusal.ts        a page's error → signedOut, notOperator, startAgain or failed; an action's in words (wordsOf),
                    the server's UTC times in the operator's own (localized)
  useCall.ts        one call's state, run on mount
  useAction.ts      one read's state, run on a click
  useRoute.ts       the page shown and what is open on it, kept in the address bar
  useWide.ts        whether the window is wide enough for the sidebar
  usePageVisible.ts whether the tab is shown
  useDarkScheme.ts  whether the dark scheme is on, for a canvas that reads its colors as it draws
  navigation.ts     MENU, the pages by group, the route (routeOf, hashOf) and OpenPlayer
  countries.ts      a country id's name and its flag
  flags.ts          the server's list of flags, read at load and by the Flags page, which names a flag opened since and
                    gives a sold flag's picture
  main.tsx          takes the sign-in callback and a connector's request off the address bar, then renders App
  App.tsx           the callback, or WhoAmI first, then the page the menu picked
  signIn/           callbackOf, SignInButton, SignInCallback, SignOutButton
  connect/          ConnectPage (Allow or Deny an app), pending.ts (the request kept for the tab)
  components/       Menu and Sidebar, ConfirmAction, FailureNotice, AccountId and Copyable (with Copy), Country, FlagMark (a
                    flag's emoji, a sold flag's picture or a grey flag), CountryField,
                    Fact, Person, When and Ago, Fold (a panel read once opened), ChangeList (what a form changes);
                    time.ts (a time in words, a span of two, as a datetime-local holds it, the operator's time zone)
  boards/           the boards: boards.ts (BOARDS by name, boardOf), places.ts (a board's places, read once by name),
                    capitals.ts (each country's capital at its tile),
                    lattice.ts (each tile's position, rebuilt from the board's runs), scale.ts (a number tuned on
                    earth-1, read at a board's spacing); testBoards.ts (earth-1 and fixture-1, for the tests)
  home/             Home: the tiles, ActivityCharts, Heatmap, CountryTable, PlayingNow, over ledger.ts (the counts, and
                    the board the ledger is on), heat.ts (the picture) and activity.ts; useWidth.ts (a chart's width as
                    it resizes)
  support/          Ideas and bugs: NotesView (the inbox), NoteList, NoteDetail, AnswerPanel, Attachments, actions.ts
                    (an answer and a close), notes.ts (a note's conversation and where it stands), labels.ts (names,
                    outcomes, device lines, sizes)
  chat/             Conversation and TheirMessages, which reports/ and players/ show, over chatPages.ts (a page of
                    ListMessages, the parties a report names) and theirConversations.ts; said.ts (what a message said,
                    in words)
  reports/          Reports: ReportsView (the inbox), ReportList, ReportDetail, DecisionLine (a report's outcome, time and
                    comment), VerdictPanel over verdict.ts, actions.ts
  players/          Players: list, PlayerName, PlayerView, Standing, ProfilePanel, the tools on one player, actions.ts,
                    HistoryPanel over history.ts (an account's moderation history: the chat's and the renames);
                    caller.ts (an account or a scope, from what was typed), charges.ts (a grant and what is held, in
                    words), durations.ts (the mute and ban choices, a span in words)
  graffiti/         Graffiti: GraffitiView (the inbox), GraffitiDetail (one, with its painter's Ban and Revert),
                    GraffitiPicture over picture.ts (a drawing as the globe shows it, each tile its cell), graffiti.ts
                    (the list, the painters' names and standings, the boards' places, a graffiti's key); testPlaces.ts
                    (a patch of a board's lattice, for the tests)
  map/              Download a replay, actions.ts (the file, and the eras it is on); download.ts (a file saved from
                    memory)
  clips/            Clip stats: ClipStatsPage, actions.ts (the CSV's file, every row of a window), table.ts (its columns)
  lab/              The Lab: LabPage over readout.ts (each post at 7 days, each experiment against its control, the
                    arrivals' days)
  videos/           Videos: VideosPage (the list), UploadPanel, actions.ts (the upload's steps, copyText)
  flags/            Flags, the sold flags: FlagsPage (the list), OpenPanel (open one), SoldCard (one, with Add days,
                    Rename and Close), actions.ts (the open's steps, the refusals, days left, the order), pin.ts (a
                    capital as typed, in words, as a globe link, and the great-circle distance)
  submissions/      Flag submissions: SubmissionsView (the inbox), SubmissionList, SubmissionDetail (one, as it is
                    judged), FlagFigure (a picture as drawn and on a tile), CapitalMap (the capital on a planisphere,
                    the other capitals, zoom, pan and the capital moved) over mapView.ts (the view: region, world, zoom,
                    pan, a place under a point) and mapLand.ts (the board's land in pixels, a box at a time, the
                    whole world once a board), SubmissionVerdict (Approve or Reject, at the capital moved),
                    OpenSubmittedPanel (an approved new flag opened), actions.ts (the calls and their refusals),
                    submissions.ts (the lists, the filters, the next one, the board's land and capitals, the capitals
                    near one and the flags of the same name, the photo warning)
  promo/            Promo, the battle teasers: PromoPage, BackgroundsPanel (the library and its uploads), CardsPanel (a set's cards,
                    edited and previewed), actions.ts (the upload's steps, the refusals, each set's values, a card's
                    problems), fill.ts (a card's words filled as the server fills them)
  ops/              Reconcile titles, Link Discord members (discordMembers.ts, its words), Game rules, Box & quiz rules
                    (BonusRulesPanel: three RuleSetsPanels, the boxes', the bonuses' and the quizzes'), Anti-bot rules (a form read off
                    the proto), Mission rules (missionRules.ts, the four numbers), Rule schedule, Seasons, Timeline;
                    cutoff.ts (the daily cutoff: when days end, a battle off it); rules.ts (a planet set in words and as
                    the form edits it), ruleBook.ts (a module's set as RuleSetsPanel edits it, by a book: its frozen
                    switch, its groups, its set from protojson), boxRules.ts, bonusRules.ts and quizRules.ts (the boxes' book, the bonuses' and
                    the quizzes', and each set in words), RuleFields.tsx (the forms' switch, value and board fields) over
                    ruleValues.ts (a value as typed, sent and read; a pace in words), antibotRules.ts (the antibot
                    form's fields, read off the proto), overlay.ts (a set's JSON over default's), schedule.ts (the
                    schedule's rows and what each module plays), seasons.ts (the seasons' rows and form), timeline.ts
                    (the Timeline's lanes and instants)
  gen/grpc/         generated from ../../proto, committed
  gen/map/          every board's manifest, runs and capitals, copied from ../../map/boards, and boards.ts, the table
                    of them by name; committed
  testdata/boards/  fixture-1, the tests' second board, read by the tests alone and never bundled
```

**An account is shown by its id, with Copy**, beside its name, wherever a tool takes it: a name can change hands,
and the id is what the operator tools take. A report shows each id short (its first group, the whole on hover), and
Copy copies the whole. **A list is the exception**: its rows show the flag and the name alone, and the id is on the
page each opens.

Plain CSS in `src/index.css`, tokens on `:root`, dark from `prefers-color-scheme`. The page is up to 1280px wide, so a
list and the one open sit side by side on a wide screen. **It must work on a phone**: 16px gutter, and nothing may
scroll sideways, so every id and text wraps (`overflow-wrap: anywhere`), and a table hides its wide columns
(`col-wide`) below 720px, or turns each row into a card (the rule schedule, `.schedule`).
