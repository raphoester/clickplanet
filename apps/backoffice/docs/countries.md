# A country on the page

- **A country is typed by its id** (`CountryField`, the Players page's filter), and the page names it below ("dz" →
  Algeria) with `Intl.DisplayNames`, plus the game's own ids no standard names (`countries.ts`: England, Scotland,
  Wales, Brittany). It is a hint: the server holds the list of countries and refuses any other.
- **A flag opened since the build is named by the server's list** (`flags.ts`): `flags.v1.AdminService/ListFlags`, read
  once when the console opens, since the page has no stream to say it changed, and again by the Flags page on each
  read (see [Flags](flags.md)), so a flag opened there is named everywhere at once. Its id is `r-` and ten letters or
  digits, so the field takes 12. The console draws again once the list is read. Until then, and for any id the page
  does not know, the name is the id in capitals. A server with no flags module lists none.
- **A game's own flag is an emoji** (`flagOf` in `countries.ts`, `FlagMark`, `Country`), never the game's flag atlas,
  which is the frontend's: the regional indicators of an ISO code, the tag sequences of England, Scotland and Wales, and nothing
  for Brittany, which Unicode has no flag for. Windows draws the two letters instead, so a flag always goes beside
  its country's name.
- **A sold flag is its picture** (`flagPictureOf`, `img.flag-picture`), the one the list gives, at the height of an
  emoji flag.
- **Any other id draws a plain grey flag** (`blankFlag`, `.flag-blank`): a flag opened since the list does not show
  yet, or an id the page does not know. `flagged` says whether `FlagMark` draws anything.
