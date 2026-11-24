# A country on the page

- **A country is typed by its id** (`CountryField`, the Players page's filter), and the page names it below ("dz" →
  Algeria) with `Intl.DisplayNames`. It is a hint: the server holds the list of countries and refuses any other.
  England, Scotland and Wales had names of their own here until they left the build to be sold (2026-11-24): the list
  names them now.
- **A flag opened since the build is named by the server's list** (`flags.ts`): `flags.v1.AdminService/ListFlags`, read
  once when the console opens, since the page has no stream to say it changed, and again by the Flags page on each
  read (see [Flags](flags.md)), so a flag opened there is named everywhere at once. Its id is `r-` and ten letters or
  digits, so the field takes 12, or the code of a flag that left the build to be sold (`xb`, `gb-eng`, `gb-sct`,
  `gb-wls`): Intl names
  private-use codes as tests (`XB` is "Pseudo-Bidi"), so the page takes no name from it for an `x` code but Kosovo's. The console draws again once the list is read. Until then, and for any id the page
  does not know, the name is the id in capitals. A server with no flags module lists none.
- **A game's own flag is an emoji** (`flagOf` in `countries.ts`, `FlagMark`, `Country`), never the game's flag atlas,
  which is the frontend's: the regional indicators of an ISO code. Windows draws the two letters instead, so a flag
  always goes beside its country's name.
- **A sold flag is its picture** (`flagPictureOf`, `img.flag-picture`), the one the list gives, at the height of an
  emoji flag.
- **Any other id draws a plain grey flag** (`blankFlag`, `.flag-blank`): a flag opened since the list does not show
  yet, or an id the page does not know. `flagged` says whether `FlagMark` draws anything.
