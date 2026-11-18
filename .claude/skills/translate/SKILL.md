---
name: translate
description: Translate the game page's catalogues (its lines in apps/frontend/src/i18n/locales/en.json, each board's landmass names, "in" each flag's country) into every language it has, with Haiku subagents and no API key. Use after an English line is added or changed, a board gets new landmass names, a flag or a language is added, or when asked to translate the page.
---

# Translate the game page

Translation is a Claude Code job: Haiku subagents translate, and no code calls an API. A small Go command,
`apps/backend/cmd/translations`, does the bookkeeping with no network: it finds what is new or changed, writes it in
batch files, then checks the answers and writes them with their fingerprints. The page's side (the files, a line's
syntax, the fingerprints) is in [`apps/frontend/docs/i18n.md`](../../../apps/frontend/docs/i18n.md).

Use a new folder of the session's scratchpad for each round, `<scratchpad>/translations/<n>`: `plan` refuses a folder
that is not empty, so no answer of an older round is taken.

## 1. Plan

```bash
cd apps/backend && go run ./cmd/translations plan -work <folder>
```

It prints, for each catalogue (`lines`, `landmasses/<board>`, `countries`) and language, the lines to translate, the
lines kept, the lines written by hand (no fingerprint: they are kept and get one), the landmass names another board
already gives, and the keys English no longer has. It writes `<folder>/<catalogue>/<tag>.<n>.json`, at most 400 lines
each (`-batch`). `-languages fr,de` limits it.

- **Nothing to translate**: still run step 4 when a line was written by hand, to fingerprint it.
- **A landmass line carries Natural Earth's names** when the page's cache has them
  (`apps/frontend/node_modules/.cache/natural-earth/`). The plan says when it lacks them: `npm run landmassNames`
  downloads them.

## 2. Brief each language

- **The page's lines** (catalogue `lines`) need the glossary, `apps/frontend/src/i18n/glossary.json`, and the plural
  categories of the language, as the page's catalogue test checks them:

  ```bash
  node -p 'const l = "pl"; [new Intl.PluralRules(l).resolvedOptions().pluralCategories, new Intl.PluralRules(l, {type: "ordinal"}).resolvedOptions().pluralCategories].map(String).join(" / ")'
  ```

- **Landmass names and countries** need neither.

## 3. Translate: one Haiku subagent per batch

Start them all in one message, so they run at once: the Agent tool, `model: "haiku"`, `subagent_type:
"general-purpose"`, a prompt like this one (absolute paths):

```
Translate a batch of ClickPlanet's lines into <language name> (<tag>).
1. Read <repo>/.claude/skills/translate/instructions.md and follow it.
2. Read the batch <folder>/<catalogue>/<tag>.<n>.json.
3. (catalogue "lines" only) Read the glossary <repo>/apps/frontend/src/i18n/glossary.json and use its "<tag>" word
   for each term. <tag> needs the plural categories <cardinal> and the ordinal categories <ordinal>.
4. Write your answers to <folder>/<catalogue>/<tag>.<n>.answers.json: one JSON object, every key of the batch with its
   translation, nothing else. Edit no other file.
Then answer in one line: how many keys you wrote.
```

## 4. Apply

```bash
cd apps/backend && go run ./cmd/translations apply -work <folder>
```

It writes every catalogue in `apps/frontend/src/i18n/`, in `en.json`'s order, each line with the fingerprint of its
English. It refuses an answer that is empty, does not parse, or loses a slot or a tag, and fails listing every line
still not translated. For those, plan again in a new folder and translate only those batches, or write them by hand.

## 5. Test

```bash
cd apps/frontend && npm test -- src/i18n
```

`catalogue.test.ts` checks every key, slot and plural category of each language. A board translated for the first
time comes off `AWAITING_TRANSLATION` in that test.

## 6. Review a sample, French first

Read the French against the English: the glossary's words, "tu", the narrow no-break space (U+202F) before `?`, `!`,
`:` and `;`, a country's article ("au {flag} Brésil", never "sur Argentine"), a place as an atlas names it. Then a few
lines of each other language: no English left, no line cut, the right script. Fix a line by hand in its file and leave
its fingerprint: the English did not change, so it is never planned again.

## 7. Commit

The files under `apps/frontend/src/i18n/` with their fingerprints, in one commit.
