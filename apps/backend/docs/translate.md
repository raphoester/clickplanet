# The page's translator

**`cmd/translate` (`make translate`) is our own Lokalise**: it keeps the game page's translations in the repo, in
`apps/frontend/src/i18n`, and asks Claude only for the lines that need it. The page's side (the catalogue, the
syntax of a line, how the language is chosen) is in the frontend's [Translation](../../frontend/docs/i18n.md).

```bash
ANTHROPIC_API_KEY=... make translate                      # every language the page has a file for
make translate ARGS="-plan"                               # what would be sent; asks nothing, needs no key
make translate ARGS="-languages fr,de -model claude-opus-5-5"
```

## What it sends

- **Two catalogues**: the page's lines (`src/i18n/locales/en.json` → `locales/<lang>.json`), and the names of the
  landmasses that are not a country's main one (`static/landmassNames.json` → `landmasses/<lang>.json`).
- **The languages are the files**: each `locales/<lang>.json` beside `en.json`. A new language is an empty `{}` file
  named after its tag, then a run, then the page's `LANGUAGES` (`src/i18n/languages.ts`).
- **Only a line that is new, or whose English changed, is sent.** Beside each language's file,
  `src/i18n/fingerprints/<lang>.json` (and `fingerprints/landmasses/<lang>.json`) holds, for each key, the first 16
  hex digits of the SHA-256 of the English it was translated from. A line whose English is the same is never sent
  again, so **a fix made by hand stays**. A line with no fingerprint was written by hand: it is kept and fingerprinted,
  and the run needs no key when nothing else is to send. A key English no longer has is dropped.
- **Each line carries its context**: its note from `src/i18n/notes.json`, when the code does not say what it means
  (a lone "Off", "Big"), and the first lines of the page's code that show it (`file: line`), found by searching
  `src/` for the quoted key. A landmass carries Natural Earth's own names for it (below).
- **Every call carries the glossary and the tone.** `internal/translation/glossary.json` holds the game's words
  (tile, flag, shield, fortify, enclose, bomb, spread, refill, box, quiz, season, battle, Final Battle, mission,
  title, front, capital, knock out…), what each means, and the word chosen in each language.
  `claude_translator/instructions.md` holds the tone (short sentences, a teenager's game words, the informal "you" of
  each language, neutral Spanish, Brazil's Portuguese) and how a line is written (its slots, plural categories,
  tags). Both are sent first and cached; the lines go 60 to a call, 4 languages at once.
- **The model is a flag**, `claude-sonnet-5-5` by default: the page's ~750 lines in 8 languages cost cents. The run
  prints the tokens it spent.

## What it writes

- **An answer that loses a slot or a tag, or does not parse, is refused** (`translation.SlotsDiffer`, the same
  subset of ICU as `src/i18n/message.ts`): it is not written, the run says which and fails, and the next run sends it
  again. A count may become a plain `{count}` in a language with one form for every number. The page's catalogue test
  then checks every plural category each language needs.
- The files keep `en.json`'s order, two spaces deep, `<` and `&` as they are.

## The landmasses and Natural Earth

`static/landmassNames.json` is made by the page's `npm run landmassNames` from Natural Earth (`v5.1.2`), in English.
Natural Earth names most of those places in French, German, Spanish, Portuguese, Polish, Turkish and Indonesian (not
Czech), but as a gazetteer does: "Gespanschaft Split-Dalmatien", "Kabupaten Split-Dalmatia", "splicko-dalmatyńska",
Portugal's "Ilhas Jónicas". So its names are not copied: `natural_earth` reads them from the same layers (cached
where the page's script keeps them, `node_modules/.cache/natural-earth/<tag>`, downloaded once), and each landmass's
line carries the ones every place of that English name agrees on, as the reference the model starts from. The
instructions tell it to drop the kind of division the English does not name, write a name and not an adjective, use
Brazil's spelling, and keep a trailing number ("Ionian Islands 2").

## Built for what comes next

The quiz (7,196 questions) and the titles are next. The packages take any keyed lines with a context, not the page's
files: `translation` plans and checks, `claude_translator` asks, `catalogue_files` reads and writes a catalogue. A new
source is a package beside `page_source` and `landmass_source`, and a catalogue in `page_catalogues` or a job of its
own.

## Code

- `cmd/translate`: `flags`, then `page_catalogues.Load` (the plans), `display.Plan` with `-plan`, or the translator,
  `page_catalogues.Translate` (each language saved as it comes back) and `display.Outcomes`.
- `internal/translation`: `PlanOf`, `Translate`, `FingerprintOf`, `SlotsDiffer`, the glossary.
