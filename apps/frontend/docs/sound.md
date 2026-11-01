# Sound

`src/app/sound/` holds it. **There is no audio file**: every sound is built
from oscillators and noise with the Web Audio API at the moment it plays, in
`synths.ts`. Tuning a sound is changing numbers there.

- `soundPlayer.ts` — `createSoundPlayer`: settings check, a per-sound
  `MIN_GAP_MS` (fast clicks and a busy chat would otherwise be one long buzz),
  nothing in a hidden tab. Its context, synths and clock are injectable, so it is
  tested without audio.
- `useSound.ts` — the settings in `clickplanet-sound-settings`
  (`domain/soundSettings.ts` parses them; a sound added later starts on) and
  the one player. **`play` never changes identity** and reads the settings
  through a ref: `useGlobe` rebuilds the globe when an option changes, and a
  toggle must not do that.
- `SoundSettingsPanel.tsx` — the switches, in the Settings place
  (`settings/SettingsPlace.tsx`). Turning a sound on previews it.

**Audio is locked until a gesture.** The `AudioContext` is only created by the
first `pointerdown`/`keydown` on the window, so a bonus box or a chat message
before the player has touched the page is silent, by design of the browser.

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
click's;
the chat in `ChatPanel` for a message that is not yours. **Your own message is
filtered on your account as well as on `mine`**: its broadcast can arrive before
the send answer that fills `mine` in. A guest's account is only known once a
message of its own came back, so its first can still ping.

## The leader's anthem

The one sound that **is** a file. `src/app/anthem/` plays the national anthem of
the country first on the board, in the order picked, on a loop, with a player in
the leader's frame on the board. So the frame and the music always name one
country: the season's leader under Season, the country with the most tiles under
Territory or with no race.
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
  so `Viewer` hands `useAnthem` no board until the map is loaded: the board is
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
