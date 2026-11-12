# Player videos

**A player makes an 8 s video of its flag's day in its own browser, and posts it.** The video shows the player's
address, which brings a viewer into the game on that flag and keeps who recruited them (see [Arrivals](arrivals.md)).
It is a 2-week test (from 2026-11): do players post videos that bring new players? No event, call or log counts a
video made, shared or saved. There is no reward in the game (see [Delivery](#delivery) for the reposts).

## Who can make one

- **Everyone who clicked today, guests too.** The browser keeps the day of its last accepted click
  (`clickplanet-video-clicked`, the player's own day).
- **Only where the browser encodes H.264**: `encoderConfig` (`src/video/job.ts`) asks
  `VideoEncoder.isConfigSupported` for High, then Main, then Baseline at level 3.1. No yes, no tile.
- **Not during an arena battle**, whose map is not the world's. No read tells the page that a map is an arena's (the
  backend's [Arena battles](../../backend/docs/arenas.md)), so a battle that plays the knock-out is taken for one.
- **Not on CrazyGames**: the address leads away from it.

## Where it is offered

- **"Make my video", the first tile of Share** (`app/video/VideoTile.tsx`, in `SharePlace`), above "Take a picture".
- **Three moments**: the day reveal of a day the flag won (`wonTheDay`) gets "Make my video"; the flag losing a fifth
  or more of a country gets a moment of its own (`VideoMoment`, the announcement `video`, once a day,
  `clickplanet-video-loss`), with the headline; "You are in a clip!" gets "Make yours".
- **Red dots**, one on the tile and one on the Share tab (the menu's rail and the phone's tab bar), in `--action`. They
  light at most once a day, only when the flag gained or lost a tenth of a country in the 24 hours, at the first visit
  of the day or the first of the three moments, whichever the page reads first (`dotsAfter`,
  `src/domain/video/dots.ts`). Only a tap on the tile puts them out. The browser keeps them in
  `clickplanet-video-dots`.

## The 24 hours

- **Planet's `GetRecentMapHistory`** (the backend's [The map's last 24 hours](../../backend/docs/recent-map-history.md)):
  49 frames, 30 minutes apart, each change with the account that made it. About 830 KB.
- **Read once the map is up and the page idle** (`useVideoOffer`, `requestIdleCallback`), and only for a player who
  can make one. It is a GET the browser's cache serves until the next step; the video screen reads it again only once
  a newer step is out (`stale`). Never on a render. The borders blob is read beside it, for each tile's country.

## The story

All of it is pure and tested, in `src/domain/video/story.ts`:

- **The country**: the one real country (the ground of `countryOfTile`) where the flag gained or lost the most tiles
  between the first frame and the last. Under a hundredth of that country is a quiet day: then it is the country where
  the flag holds the most, weighed by its share, so 62% of Poland comes before 1% of Russia.
- **The headline**, a fixed English sentence: "POLAND TOOK 40% OF GERMANY", "POLAND IS LOSING GERMANY. HELP", "POLAND
  HOLDS 62% OF POLAND". A share is never rounded up, and never 0% for something. Names as the clips say them
  (`src/clip/names.ts`: CLDR's English name, with its article).
- **The allies**: the accounts whose changes won the flag the most tiles of that country, the maker left out. Never an
  account that took from the flag.
- **The names**: clips' `GetShownNames` for the maker and the first 30 allies. A name it leaves out is not shown: three
  allies at most, among those shown. A guest, and a player whose name is not shown, has no name on the video.

## The address

`shareAddress` (`src/domain/shareCard.ts`) puts the username in the address only when `GetShownNames` answers that very
name for the player: `clickplanet.lol/pl/kutza`, else `clickplanet.lol/pl`. The recruit link, the picture and the
video all follow it. `useShownName` reads it, and once more 30 s later when the name is not shown yet: a new name is
judged within seconds.

## The video

- **Drawn in a page of its own**, `/video` (`video.html`, `src/video/main.ts`), in a hidden frame
  (`app/video/makeVideo.ts`). The page has its own clock and its own globe, so the game's globe and clock run on. It
  keeps no storage of the game's: `isolate.ts` gives it an empty one, or its globe would paint the player's ghosts.
- **The frames are drawn one by one on the virtual clock** (`src/clip/virtualClock.ts`), never in real time: 240 frames,
  720x1280, 30 fps. The globe plays the 24 hours as a replay (`ReplayBackend.ofHistory`). The camera is the clips'
  front camera (`cameraOf`): it dives where the flag fought in the country and comes back out over it at the end.
- **The look is the official clips'** (`src/video/overlay.ts`, on a canvas, in their 540-wide sizes): the headline at
  the top with the flag, the flag's share of the country and its change, the address small, and a track. They stay
  out of the apps' buttons: never the bottom fifth, never the right edge. After 6 s, the end card: the flag, the
  maker's name, the allies, and the address big, in Rubik so the name keeps its case. A name Luckiest Guy cannot draw
  sets every display word in Rubik Black.
- **Encoded by WebCodecs** (`VideoEncoder`, H.264, 5 Mbit/s) and **muxed by Mediabunny** (MPL-2.0) into an MP4 with
  its index first.
- **The sound is the flag's anthem** from the game's recordings (`static/anthems/`, AAC LC in M4A, see
  [Sound](sound.md#the-leaders-anthem)), from its best highlight as the clips start it. Its AAC packets are copied into
  the MP4 as they are: no audio encoder, since iOS Safari's is broken. So it cannot fade out: it stops at 8 s. A flag
  with no recording (the clips' extra anthems in `scripts/clip/anthems/` are not served), or an anthem that cannot be
  read, gives a silent video, never no video.

## Delivery

After the video is made the player taps again, which `navigator.share` needs (`videoDeliveriesOffered`): a phone that
shares files gets Share, an iPhone Share and Save (TikTok is often missing from its sheet), a computer Save. A guest
also gets "Sign in to put your name on it".

**Under the video: "Tag @clickplanetlol on TikTok. We repost the best."** The handle is `TIKTOK_HANDLE` in
`links.ts`, selected whole by a press. The operator reposts by hand with TikTok's Repost button: the video stays the
player's, and a repost is the one thing a player gets for posting. Never download and upload a player's video again:
TikTok takes it for unoriginal content and keeps it off the For You feed. Nothing reads the tags: no API lists a
TikTok account's mentions for the posting tool. The line is not in the shared text, which goes to every app.

## Welcome line

A browser that came by an address with a name holds the recruiter's account in its landing. Once, when the map is up,
it reads that account's shown name and tells "kutza called you to fight for Poland" on the news line
(`useWelcomeLine`, `WelcomeNews`). No shown name, no line. `clickplanet-recruit-welcomed` keeps that it was read.

## In fake mode

`VITE_FAKE_BACKEND=1 npm run dev`: `FakeRecentHistory` makes up 24 hours that end on the fake map, each change by one
of the fake takers, and `FakeShownNames` shows every name the fakes know but a guest's. The tile shows once a click of
today is kept: click a tile, or set `clickplanet-video-clicked` to today.

## Not measured

On this project's Mac (Nov 2026), a video took 2 to 4 s in the dev server. A phone's time is not measured.
