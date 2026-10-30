# Sharing the globe

The game's own map is the marketing material, so the globe can be photographed
and the picture taken out of the browser, and a player can hand out a link that
brings a newcomer in under its flag. `src/app/share/` holds both:
`SharePlace.tsx` is the Share place, a tab of the menu and of the phone's tab
bar. Its "Take a picture" tile takes the shot,
`takePicture.ts` captures and composes it,
`drawShareCard.ts` draws the card, `SharePreview.tsx` shows it, `ShareActions
.tsx` is the row of buttons under it and `deliverShare.ts` is what they do.
`useSharePicture.ts` holds the one picture there is at a time.
`src/domain/shareCard.ts` holds everything decided before a pixel is drawn.

**The camera is a tile in Share, not a button on the globe.** It used to sit
on the canvas, bottom-left, beside its subject; it was one of six things over
the globe on a phone, and it is pressed rarely. On a phone the press closes the
sheet first, so the globe is what the preview shows the player framed.

**The recruit link is under it** (`RecruitLink.tsx`): "Recruit for France", the
link `clickplanet.lol/?f=fr` of the country played (`shareLabel`), and one
button. The link needs nothing new: `?f=` is the flag a shared picture already
carries, and a newcomer who opens it plays for that country.

- **A phone gets its share sheet, a desktop a copy** (`linkDeliveryOffered`,
  `deliverLink` in `deliverShare.ts`). The sheet takes the link and the sentence
  apart (`shareHeadline`), so no target shows the link twice; a copy is the link
  alone. A sheet that fails copies; a sheet the player closed does nothing.
- **A refused clipboard falls back to copying a selection**
  (`document.execCommand`): an in-app browser, which is where a link from
  TikTok or Discord opens, often refuses the clipboard API.
- **The link is on its own line, the button under it.** Side by side, the link
  was cut on a phone 360px wide, and "Copied!" is wider than "Copy link".
- **The link is text, not a field**: `user-select: all`, so a press selects the
  whole of it where nothing else copies.
- `useDelivery.ts` is the button's states for this and for the picture's
  buttons: working, done for 2.5s, failed, and a closed sheet is no outcome.
- **The button is `RecruitButton.tsx`**, so the frozen dock sends the same link
  (see [A frozen map](backends.md#a-frozen-map)): there it is "Invite" with the
  flag, and the sentence is `breakHeadline`, the next season on a blank map.

**Its label never changes.** What answers the press is the preview opening.
Working is said by the tile dimming (`aria-busy`, disabled).

**Pressing it opens a preview, and the preview is where the choice lives.** The
framing is the player's — the camera takes the globe at whatever angle and zoom
they left it — and that is the one thing they cannot check after the fact. A
capture that fails opens the preview too, saying so: a camera button that does
nothing visible is a button pressed again and again.

**`preserveDrawingBuffer` is deliberately off**, which is why the capture is
where it is. Setting it would have the driver keep a second copy of the buffer
for every frame of every session, permanently, to serve a button most players
press rarely or never. Instead `globe.ts` runs the read from an `afterRender`
hook inside the animation loop, in the same tick as the `render()` that filled
the buffer, and `Globe.capture()` hands back a promise that settles on the next
frame. A read one tick later comes back blank. Two consequences worth knowing:

- **Reading the canvas rather than an offscreen target is also what keeps the
  colours right.** three applies the output colour space conversion only when it
  renders to the canvas — a `WebGLRenderTarget` that is not an XR one is forced
  to `LinearSRGBColorSpace` — so the same scene drawn into a target comes back
  visibly different from what the player was offered.
- **A hidden tab has no frames**, so a capture started and then backgrounded
  sits on "Drawing…" until the tab is looked at again. It settles by itself.

**The card is composed, never screenshotted.** The DOM over the globe is a
translucent panel with a scrolling leaderboard in it; what is good to use is a
poor picture. The badge is drawn from the same numbers the menu is drawn from,
out of the same flag atlas, in hundredths of the card's shortest edge so it
reads the same on a phone in portrait as on a wide desktop. **Its rank is the
one the board shows**: `Viewer` hands `shareStats` the lines of the order picked,
so a card taken under Season carries the season's rank beside the tiles held.
The season ranks a country with points and no ground, which the card says as
"RANK #2 · NO TILES"; a country on no board at all is still "NO TILES YET".

**The card carries the mark across the top** — the same logo and wordmark the
menu header flies — with the link at the other end of that line, and the
player's badge at the bottom.

**The link switches whoever opens it to its flag.** `main.tsx` reads `?f=`
with `sharedCountry`, before the first render, and the game starts on that
country over the one in storage, then stores it as if it had been picked. An
unknown code is ignored. The parameter is then taken out of the address bar,
as the sign-in code is: left there, a reload would undo a flag the player has
switched since. **The link a player sends also carries `?s=rc`**, so an
account made by a browser that came by it counts as a recruit (see
[Arrivals](arrivals.md)). The words drawn into the picture leave it out
(`shareLabel`): longer, they meet the logo on a portrait card.

**The link is drawn into the image**, not only attached to it: a picture is what
survives being reposted. It is drawn in the text face (`--font-text`, Rubik)
rather than the title face, which has no lowercase — a query parameter reading `?C=PS` is a link that
does not work for whoever retypes it — and it sits on the masthead's line rather
than over the badge, so a long country name never has to share a width with it.

**Anything drawn beside a title is aligned on the capitals, not on the box.**
Luckiest Guy carries far more ascent than its capitals use, so `align-items:
center` and canvas's `textBaseline: "middle"` both leave the letters riding above
whatever is centred next to them. `titleFont.ts` holds that one fact and what it
costs; `CountryFlag` is the component that settles it for the DOM and every flag
beside a name goes through it, while the card measures the cap band off the face
at draw time and centres on that. The badge's panel is sized from the same
measurement rather than from font sizes, which is what makes its padding even —
a row measured in `px` of font is mostly leading. **A flex `margin-top` doing
this correction has to be twice the rise**, because centring applies to the
margin box; getting that wrong left the camera icon exactly half-corrected.

**The capture is the drawing buffer**, at a ratio of 1, or at the screen's
capped at 2 with HD graphics (see [CSS pixels in, drawing-buffer pixels
out](pixel-ratio.md)): a phone captures around 390×844,
or 780×1688 with HD graphics, and a ratio-1 desktop its CSS size. `cardSize` lifts a small one to a
short edge of 720 — the globe softens a little and the flag and the counts stay
crisp, which is the half anyone reads — and caps the long edge at 2400 so a
share sheet will still take the file.

**And the card is the middle of the frame, not all of it.** A phone's frame is
a 1:2.2 column that every timeline either shows as a sliver or crops for you;
`cropToAspect` brings the shape back inside 9:16 … 16:9 first, centred, because
the globe is centred — the camera looks at the origin. The portrait limit is the
loosest of the standard shapes on purpose: at rest the sphere's diameter is the
viewport's *height*, so on a phone it is already wider than the screen and every
row cropped is a row of planet. The tighter 4:5 a feed prefers takes nearly half
the frame.

**The player picks the delivery; nothing picks for them.** `deliveriesOffered`
reads once, when the menu mounts, and puts one button on screen per way this
browser actually has of letting go of the file — a phone gets `Share`, a desktop
gets `Copy` and `Save`. It is never empty: a download needs nothing of the
browser.

This replaced a ladder that tried the three in turn and reported whichever
answered first, and both halves of that went wrong in the first minute of real
use. The desktop share sheet reported success and posted the text with no
picture. A clipboard write the browser had quietly refused came back as a
download, so the button said "Saved!" on a press that asked to copy. **What is
offered is only what this browser can do, and what happens is only what was
asked for** — which is also why a refused copy reports a failure instead of
falling through to the download sitting an inch away from it.

**The share sheet is a phone's button**, and that judgement is the one thing
there that is not a feature test. On a phone the sheet *is* how you send a file
somewhere and it carries one; on a desktop it is a shim over the OS share
services, and Chrome on macOS answers `canShare({files})` true for services that
then keep the text and drop the image — measured, into Telegram, which posted
the sentence and no picture. `canShare` is asked with a stand-in `File`, since
it judges the kind of thing it is handed rather than the bytes. A share sheet
the player *cancels* delivers nothing, rather than handing them a file seconds
after they said no.

**The clipboard carries the picture and nothing else**, for the same reason
wearing a third hat: handed an item with `image/png` *and* `text/plain`, a chat
window pastes the sentence. That is why the link is drawn into the image — it
has nowhere else it has to be, so the copy has one fewer way to be misunderstood.

**The delivery buttons live in the preview's footer**, so the picture is on
screen while the player picks what to do with it. `Modal` takes a `className`
for the panel — `SharePreview.css` widens it past the 360px `Modal.css` sizes a
column of text to, drops the scroll fade that would veil the bottom of the card,
and fits the picture to the room between the header and the buttons rather than
capping it in `vh`, which left a hand's width of empty panel under a portrait
card on a phone.
