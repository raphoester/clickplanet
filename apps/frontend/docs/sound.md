# Sound

`src/app/sound/` holds it. **There is no audio file**: every sound is built
from oscillators and noise with the Web Audio API at the moment it plays, in
`synths.ts`. Tuning a sound is changing numbers there.

- `soundPlayer.ts` — `createSoundPlayer`: settings check, a per-sound
  `MIN_GAP_MS` (fast clicks and a busy chat would otherwise be one long buzz),
  nothing in a hidden tab. Its context, synths and clock are injectable, so it is
  tested without audio.
- `soundSettingsStore.ts` — the store `useSoundSettings` (see [Stores](stores.md)): the settings, in
  `clickplanet-sound-settings` (`domain/soundSettings.ts` parses them; a sound added later starts on).
  The switches and the anthem's controls read it and change it.
- `useSound.ts` — the one player. **`play` never changes identity** and reads the settings with
  `getState` when it plays: `useGlobe` rebuilds the globe when an option changes, and a toggle must
  not do that.
- `SoundSettingsPanel.tsx` — the switches, in the Settings place
  (`settings/SettingsPlace.tsx`). Turning a sound on previews it.

**Audio is locked until a gesture.** The `AudioContext` is only created by the
first `pointerdown`/`keydown` on the window, so a bonus box or a chat message
before the player has touched the page is silent, by design of the browser.

**A context runs only while it can be heard.** A running `AudioContext` keeps
the audio hardware awake, and the browser's audio thread with it, sound or not.

- **No context while every sound is off.** `unlock` makes one only if a sound
  is audible. Turning a sound on in the settings previews it, and the preview
  makes the context inside that tap. Otherwise the next gesture makes it.
- **The effects' context is suspended `SILENCE_MS` (10 s) after the last
  sound**, longer than the longest one (a bomb's fall, its 4.5 s tail and the
  room's echo). The next sound wakes it: `start` resumes it and drops the sound
  if the wake takes more than `LATE_MS`. Each gesture wakes it as well, because
  iOS lets a context start only inside a gesture: `resume` is called at once
  there, never after a wait. A sound that comes while a suspend is under way
  waits for it, then wakes the context.
- **The anthem's context is made only when the anthem is heard**, after a
  gesture: at the gesture itself, or when the player turns it on. Play, the
  switch and the volume are taps, and React runs the effect of a tap's update
  inside that tap, so iOS sees the gesture. **It is suspended a second after
  the anthem goes quiet** (off, volume 0, muted or a hidden tab), when the
  fade is all but over and the recording is paused. It is resumed when the
  anthem is heard again. A recording is fetched only to be heard: a new leader's
  anthem waits for that.
- **Why.** Before, every page made both contexts at its first tap and kept them
  running for as long as it was open, even with the sound off. Measured on
  2026-11-19 in headless Chrome on an M2 Max, the globe still, the page in use,
  renderer and GPU process together: with the sound on, the anthem and the chat's ping
  off and nothing playing, 11.4% of a core before (two contexts running), 9.0% after (one
  context, suspended); with the sound off, 11.3% before, 9.3% after (no context).
  A playing anthem runs its context, as it must. The fake's chat pings every 8 s,
  which keeps the effects' context awake: each sound wakes it.

Where each one fires: the click and the refusal in `globe.ts`'s click handler
(`reportClickFailure` returns whether the server refused — a transport fault is
not a "nope"); the box appearing and being caught at the same places the box
itself does; the bomb when its broadcast arrives, with the boom scheduled
`IMPACT_DELAY` later so it lands with the tiles, quieter for someone else's,
and a splash instead of a blast when the drop has no tile under it (the ocean);
your own spread click and closed shape when their broadcast comes
back, so each lands with its effect on screen. **Only the player who made one
hears it.** An enclosure carries `yours`; a spread says nothing of
whose it is, so `domain/ownClicks.ts` remembers the tiles this client clicked in
the last 3s and a broadcast on one of them, for the same country, is taken as
ours. They have no switch of their own: `switchOf` puts them under the tile
click's; a trap sprung (`trap`, a metal snap: a hard click of bright noise, the
jaws' few inharmonic partials and a low thump) on every page that sees it, full
for a trap of the page's flag or the page's own take and softer for the others,
under the bomb's switch (see [Trap](trap.md));
the chat in `ChatPanel` for a message that is not yours. **Your own message is
filtered on your account as well as on `mine`**: its broadcast can arrive before
the send answer that fills `mine` in. A guest's account is only known once a
message of its own came back, so its first can still ping.

## The leader's anthem

The one sound that **is** a file. `src/app/anthem/` plays the national anthem of
the country first on the board, in the order picked, on a loop, with a player in
the leader's frame on the board. So the frame and the music always name one
country: the first under the column head the board is sorted by (Points, Tiles or
Gained), the country with the most tiles with no race.
The recordings are the US Navy Band's (public domain); `npm run anthems` downloads
them, normalises loudness, trims the silence so the loop has no gap, and writes
`static/anthems/<code>-<hash>.m4a` plus `anthemsAsset.ts`, which pairs each file
with the anthem's title (`TITLES` in the script, written by hand: Wikidata's
labels mix titles with "National Anthem of X"). It needs ffmpeg.
Territories share their country's file (`SHARES` in the script), and so does Cyprus,
whose anthem is Greece's and which the band never recorded; a country with
no recording shows the player with its play button off.

- `domain/anthemLeader.ts` — `followLeader`: a new leader must hold first place
  for `HOLD_MS` (15s) before the music follows it. The first leader plays at once,
  so `PlayScreen` hands `useAnthem` no board until the map is loaded: the board is
  sampled while the batches arrive, and a half-loaded map's leader is not the real one.
  **The leader of another order plays at once too**: the pick keeps the order it
  follows, and the hold is for two countries trading first place, not for the
  player's own switch. A race that arrives after the map is a new order as well
  (`shownOrder`), so the season's leader does not wait under the map's.
- `anthemPlayer.ts` — two `<audio>` elements crossfade through Web Audio gain
  nodes. **Not `audio.volume`: iOS ignores it.** A muted or hidden tab fades out
  and pauses rather than streaming silence.
- `useAnthem.ts` — ties the board's lines, their order and the settings to the
  player, which is built once, so a tick or a toggle never restarts the music.
- `AnthemControls.tsx` — the player, in the board's frame for the first country (`Leaderboard`'s `anthem`): the anthem's title, then the country. Play and volume write `SoundSettings.anthem`, so
  it and the settings panel never disagree. Pressing play lifts the master switch.
  It plays the country the music follows, which holds first place `HOLD_MS` before
  it changes, so for that long after a new leader the frame shows the last one's
  anthem, under its own flag.
