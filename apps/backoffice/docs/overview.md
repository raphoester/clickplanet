# What this is

**The operator's page, at `backoffice.clickplanet.lol`.** It calls the backend's `AdminService`s, which were
reachable only over SSH before, and **every admin method has a place on it**. One page at a time, picked in the
burger menu (see [The menu](menu.md)), by group:

- **Overview**: Home, how the game is doing now and over the last hours (see [The home page](home.md)).
- **Feedback**: Ideas and bugs, the notes players send, read only, each with who sent it (see
  [Who sent a note](note-sender.md)) and its photos and videos (see
  [A note's photos and videos](note-media.md)).
- **Moderation**: Reports, which the operator acts on and closes (see [Acting on a report](reports.md)),
  and Players, who played and every tool on one player (see [The player page](player-page.md)).
- **Map**: Reassign a country, Paint random tiles, Download a replay (see [The map tools](map-tools.md)).
- **Clips**: Clip stats, the daily numbers of the clips the clips job posts, and a CSV of them (see
  [Clip stats](clip-stats.md)).
- **Ops**: Reconcile titles, the run after a deploy; Game rules, the rule sets the game plays by; and Anti-bot rules,
  the rule sets the antibot judges with (see [Ops](ops.md)).

What is left of the plan is a live chat view: `ListMessages` pages through the chat (the report and player pages
read it), and no admin method streams it.

It is a third app, not a page of the game: **it imports nothing from `apps/frontend`** and has no Three.js (see
the root's [Independence of the apps](../../../docs/app-independence.md)).
