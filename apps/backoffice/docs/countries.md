# A country on the page

- **A country is typed by its id** (`CountryField`, the Players page's filter), and the page names it below ("dz" →
  Algeria) with `Intl.DisplayNames`, plus the game's own ids no standard names (`countries.ts`: England, Scotland,
  Wales, Brittany). It is a hint: the server holds the list of countries and refuses any other.
- **A flag is an emoji** (`flagOf` in `countries.ts`, `Country`), never the game's flag atlas, which is the
  frontend's: the regional indicators of an ISO code, the tag sequences of England, Scotland and Wales, and nothing
  for Brittany, which Unicode has no flag for. Windows draws the two letters instead, so a flag always goes beside
  its country's name.
