# In a clip

The clips job posts short videos of the game on TikTok and YouTube Shorts, and each one names its top
fighters (see [Clips](clips.md) and the backend's [clips job](../../backend/docs/clips-job.md)). The game tells a player when a clip names
them, so they watch it and share it, and lets them keep their name out.

- **The notice is an announcement**, `app/clips/ClipNotice.tsx`: the title unlock's full screen (its slam,
  rays and fanfare) with a gold disc and a play icon, "You are in a clip!", the clip's headline, then one
  button per platform ("Watch on TikTok", "Watch on YouTube", in a new tab) and Close, once the reveal
  played (`TITLE_REVEAL.ready`). A watch button closes it too. It waits in the announcements queue like
  a title or a battle (see [Battles](battle.md#the-announcements-queue)), keyed `clip:<post id>`.
- **Only a player with a username is asked**: `Viewer` hands `useClipNotice` the account id of a linked
  account with a username, and nothing for a guest, who is never named. It reads
  `clips.v1.ClipsService/GetAppearances` (the posts of the last 7 days that name the caller) when that
  account is known, and again when the tab comes back (`usePageVisible`), **at most once every
  10 minutes** per account (`readDue`, `APPEARANCES_READ_GAP_MS`). A sign-out forgets the last read, so
  the next sign-in reads at once. No timer polls.
- **One notice, the newest** (`appearanceToShow`): the newest post not shown before, with something to
  watch. Closing it marks every post of that read as shown (`shownAfter`), so the older ones never
  follow it one by one. The ids are kept on the device in `localStorage` (`clickplanet-clips-shown`, the
  last 50), each read and write in a `try`; without storage the notice still shows, once per page.
- **A link is the video, or the channel until the video's address is known** (`watchLinksOf`):
  `TIKTOK_PROFILE` and `YOUTUBE_CHANNEL` from `links.ts`. One button per platform; a platform this build
  does not know is dropped at the wire (`platformOf` in `clipsBackend.ts`).
- **The switch is "Show my name in clips"**, in the Account tab under the name color, for a player with a
  username only. It follows the color: `AccountStore.setShownInClips` shows the choice at once, sends
  `SetShownInClips` and keeps what the server answers, or the old choice and a line on a refusal
  (`FailedPrecondition` without a username → `unnamed`). `GetProfile` answers it with the name.
- `backends/clips.ts` is the contract, `ConnectClipsBackend` the real one: it sends the identity token
  (`session.identity()`, no mint) and reads a server without the service, or a token refused, as no clip.
  `FakeClipsBackend` is the fake: `giveClip("France vs Turkey")` in the console posts a clip that names
  you, and `signIn()` signs the fake account in (see [Sign-in](sign-in.md)), which reads it at once.
