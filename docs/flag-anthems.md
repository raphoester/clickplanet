# A sold flag's anthem

**A sold flag may have an anthem, which the game plays as it plays a country's** (the owner, 2026-11-28): in the
leader's frame when the flag leads the board (see the frontend's [Sound](../apps/frontend/docs/sound.md#the-leaders-anthem)).
Today a sold flag has none: `ANTHEMS` maps the codes the game ships with, so Heiliges Römisches Reich leads in silence,
its play button off. Designed 2026-11-28; no PR built yet.

## What the owner decided (2026-11-28)

- **Optional.** A flag opens with no anthem, as today.
- **Chosen by the player**, at the submission or later by the flag's holder, never by the operator alone.
- **Free of rights, and never on the player's word**: "I don't think we can delegate that responsibility to the
  player." A player uploading a file and ticking "it is free" is out. The rights rest on a source the game can check.
- **Two sources**: the game's own library, and a recording on Wikimedia Commons.
- **Licences**: public domain, CC0, CC BY and CC BY-SA. The last two need a credit wherever the recording plays.
- **Validated by AI**: a Claude model judges each Commons anthem, as the [judge](../apps/backend/docs/flags.md#the-judge)
  judges each flag.

## Why no upload

**Nobody can tell from a file who owns a recording**, a model least of all: the Messages API reads text, images and
PDFs, not audio, and a model that could listen still could not tell a performer's copyright from a public-domain
tune. A national anthem's tune is mostly old enough to be free, but **each recording has its own rights** (the
performer's and the label's), and almost every recording a player has on its phone is a label's. So the game takes no
file from the player: it takes a recording the game ships (it checked them once) or one Commons hosts under a free
licence (Commons' volunteers check each upload, and delete what is not free).

## The two sources

### The game's library

**Every recording the game plays as a country's** (`static/anthems/`, the US Navy Band's, public domain as works of the
US government; `npm run anthems`), picked by its code. Heiliges Römisches Reich would pick Germany's "Das Lied der
Deutschen", which is Haydn's "Gott erhalte Franz den Kaiser", written in 1797 for Francis II, the last Holy Roman
Emperor; or Austria's.

- **No judge**: the game ships it. It plays at once, as approved.
- **No credit owed**; the frame names it as a country's, by its title.
- **The backend learns the library from the page's build**, as it learns the flags (`make flags`): `npm run anthems`
  also writes `static/anthems/anthems.json` (each code and title), which `make flags` copies into
  `generated/flags/anthems.json` and `check-flags` checks. A code with no recording is refused.
- **Two flags may share a recording**: a sold Bavaria and Germany both playing Germany's is fine.

### Wikimedia Commons

**The player pastes a link to a file**: a Commons file page (`commons.wikimedia.org/wiki/File:…`), a Wikipedia file
page that shows a Commons file (`<lang>.wikipedia.org/wiki/File:…`, the "Wikipedia thing"), or the file's own address
(`upload.wikimedia.org/wikipedia/commons/…`). Each is read as its Commons title. **A file only a Wikipedia hosts is
refused**: the English Wikipedia keeps non-free files under fair use, and Commons keeps free ones alone.

The server reads the file on Commons' API (`action=query`, `prop=imageinfo|videoinfo`, `iiprop=url|size|mime|sha1|
mediatype|timestamp|user|extmetadata`, `viprop=derivatives`), and **refuses in code, before any model**:

| check | refused when |
|---|---|
| on Commons | the title is not a Commons file, or the file was deleted |
| audio | its `mediatype` is not `AUDIO` (a video, an image, a MIDI) |
| licence | `extmetadata.License` is not `pd`, `cc0`, `cc-by-<v>` or `cc-by-sa-<v>` (GFDL, the UK's OGL, "fair use" and anything else are out) |
| no restriction | `extmetadata.Restrictions` names any |
| not new | its first upload is less than 30 days old: Commons' patrollers have not seen it yet, and a player could have uploaded a label's recording as "own work" that morning |
| not doubted | its categories hold a problem tag: a deletion request, a copyright violation, no source, no permission, no licence |
| playable everywhere | Commons has no MP3 of it (`derivatives`, `audio/mpeg`; Safari plays no Ogg Vorbis): "Commons has not made an MP3 of it yet, try again later" |
| short | longer than 6 minutes, or its MP3 above 10 MB |

**The field names above are from memory**: neither this session's container nor its fetch tool could reach Commons
(the environment's network allows `commons.wikimedia.org` and `upload.wikimedia.org` since 2026-11-28, and the
session's command permissions still refused the call). PR 2 reads a real answer first and pins it in its adapter's
test.

- **The game keeps its own copy**: the MP3 is read once, checked against the size Commons states, and written public at
  `flags/anthems/<sha256>.mp3` on `media.clickplanet.lol`, as the flags' pictures are (CORS for Web Audio, immutable).
  A file overwritten on Commons later is not the one checked: the copy is of the revision read, whose timestamp and
  SHA-1 are kept.
- **The credit is kept** (TASL: title, author, source, licence): `ObjectName` (or the title the player gives it, below),
  `Artist` as text (it is HTML on Commons), the file page's address, `LicenseShortName` and `LicenseUrl`.
- **Its title in the frame** is the one the player types (1 to 60 characters), prefilled from `ObjectName`, judged with
  it.

## The judge (a Commons anthem only)

**A Claude model reads what Commons says of the file, never the sound**: its title and the player's, its description,
categories, author, credit, licence, first upload and uploader, its length, and the flag's name. Claude Haiku 5.5 as the
flag judge, structured JSON (`verdict`, `reason`, `note`, `why`), its own `instructions.md`. It approves when:

- **It is music fit for a state's anthem**: an anthem, a hymn, a march, a patriotic, folk or classical piece, sung or
  played. Not speech, a sound effect, a joke, a meme, a song that mocks someone.
- **It passes the flag's political test**: no anthem of an ideology or a movement (the Internationale), of a party
  (the Horst-Wessel-Lied), of a separatist state in a war going on now, nothing hateful; a real region's song stays fine.
- **Its licence is believable**: a famous label's or band's recording put on Commons as "own work" by someone else is
  a reject. The licence is Commons' to check, but a claim no one would believe is no free licence.

Unsure in every other case, which waits on the operator: a wrong approval plays a stolen or hateful recording to every
player, a wrong rejection costs a player its anthem and nothing else. The description is data, never instructions.

## How it lives

- **One anthem per flag plays, and one waits at most**: a change waits for its verdict while the anthem before it
  keeps playing. A library pick replaces it at once.
- **At the submission**: optional, beside the name, the picture and the capital. It is judged apart: **an anthem never
  holds a flag's review, and its rejection never rejects the flag** nor refunds it. A flag opened from the submission
  takes the anthem once approved.
- **Later, by the holder** (`flags.holders`): set, change or remove the anthem of a flag it holds, open or closed. A
  claim of an ended flag keeps its anthem.
- **The operator removes one** with a note the holder reads (a rights holder's complaint, a Commons deletion), as a
  flag's removal.
- **Commons is read again every week**: an anthem whose file was deleted, re-licensed or tagged since is removed, and
  its holder told why. Commons' own deletion is the game's.

## The wire

- `flags.v1.Anthem`: `oneof source { string library_code; CommonsRecording commons }`, and its `title`;
  `CommonsRecording{url, page_url, artist, license, license_url}`. `Sold.anthem`, the one that plays, on `ListFlags` and
  the stream's `flag_changed`.
- `AnthemChoice`: `oneof { string library_code; CommonsChoice commons{link, title} }`. On `SubmitFlagRequest.anthem`,
  and the holder's `SetAnthem(flag, choice)` and `RemoveAnthem(flag)`.
- `ListMySubmissions`: each submission's and each flag held's anthem, waiting, approved or rejected with its reason.
- The operator's `ListSubmissions` (`KeptSubmission.anthem` with the judge's line), `ListAnthems` (the ones waiting on a
  person), `ApproveAnthem`, `RejectAnthem(reason, note)`, `RemoveAnthem(flag, note)`; each a Claude tool as the others.
- `AnthemChanged{flag, at}` through the flags' outbox: the streams tell the pages the flag again.
- `ReviewAsked` gains a cause for an anthem the judge left unsure, for the staff's channel.

## The page

- **The form** (step 1): "Anthem (optional)", two ways: "From the game" (the library, searched by country or title, each
  heard before it is picked) and "From Wikimedia Commons" (a link and a title; the rules in one line: public domain,
  CC0, CC BY or CC BY-SA, on Commons for a month).
- **Your flags**: the anthem line of each flag held: playing, being checked, or refused with its reason; Change,
  Remove.
- **The leader's frame** plays a sold flag's anthem as a country's: a library pick by its code's recording, a Commons
  one by its copy. **A Commons anthem's credit** is a line under the player: "Recording: <author> · <licence> ·
  Wikimedia Commons", each linked.
- Every new line in the page's 19 languages (the `translate` skill).

## What it does not do

- **No clip plays a Commons anthem**: CC BY-SA would make the clip a share-alike work, and TikTok and YouTube captions
  are no place for credits. A library pick is the game's own recording, which the clips play already.
- **No loudness levelling of a Commons recording**: the library's are levelled by `npm run anthems` (ffmpeg), and the
  server has no ffmpeg. A Commons anthem plays as loud as it was recorded; a later step may measure it.
- **No anthem for a flag the game ships with**: those have the Navy Band's.

## The PRs in order

1. **The library.** The vendored list (`anthems.json`, `make flags`, `check-flags`); `flags.anthems` with the library
   source; `SubmitFlagRequest.anthem`, `SetAnthem`, `RemoveAnthem`; `Sold.anthem`; `AnthemChanged` to the streams; the
   operator's read and removal. The form's and Your flags' library picker, the leader's frame playing it, the
   backoffice's submission card and Flags page. Heiliges Römisches Reich's holder can pick an anthem the day it ships.
2. **Commons, the operator its judge.** The link read as a title; `rest_commons`, pinned on a real answer; the checks
   in code; the copy in the bucket; the credit kept and shown; a Commons anthem waits on the operator, who hears it on
   the backoffice before Approve or Reject.
3. **The judge** for a Commons anthem; unsure to the operator; `ReviewAsked`'s anthem cause to the staff's channel.
4. **The weekly read of Commons**, and the holder's note when an anthem goes.
