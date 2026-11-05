# Videos

`src/videos/`, one page (`VideosPage`), in the menu under Clips, at `#/videos`. **The operator posts the season-end
videos to TikTok by hand from a phone**: they upload them from the Mac here, then open the page on the phone to
download each one and copy its caption. `clips.v1.AdminService` keeps them (`CreateVideoUpload`, `KeepVideo`,
`ListVideos`, `RemoveVideo`).

- **Upload** (`UploadPanel`): "Pick videos" takes one file or more (`video/mp4`, `video/quicktime`, `video/webm`).
  Each picked file has a name (the file name without its extension), a caption and its own Upload button. An upload
  is three steps: `CreateVideoUpload` with the file's type and size, which answers a video id and a signed link; one
  `PUT` of the file straight to the bucket (`putFile` in `api.ts`, with `XMLHttpRequest`, since `fetch` reports no
  upload progress), with the same `Content-Type`; then `KeepVideo` with the name and the caption. The button shows
  the step and the percent sent, then the card says "Uploaded as …" and the list reads again.
- **A refused step says why, and Upload tries again**: `CreateVideoUpload`'s `invalid_argument` (a type or a size the
  server refuses) quotes the server, its `failed_precondition` is "Videos are off on the server"; a failed `PUT`
  quotes the bucket's status and `<Message>`. **A refused name does not send the file again**: the card keeps the
  video id of the file the bucket holds and only calls `KeepVideo`. `KeepVideo`'s `failed_precondition` (the file
  is not in the bucket) forgets it, so the next try sends the file again.
- **The upload does not ask first**, unlike the other changes (see [Every action](actions.md)): picking a file and
  pressing Upload is the question, and Remove takes it back.
- **The list** (`ListVideos`, the newest first): each video plays in place (`preload="metadata"`, `playsinline`), with
  its name, size and date, its caption in a box with **Copy caption**, a **Download** link and **Remove**, which asks
  first. Download is a plain link: the server signs `download_url` with `Content-Disposition: attachment`, so the
  file saves under its name, on iPhone Safari too. A video the browser cannot play, or a link past its time, says so:
  Reload signs new links.
- **Copy caption** uses `navigator.clipboard`, then the old copy command; if the browser refuses both, it selects the
  caption so the operator copies it by hand.
- **`ListVideos`' `failed_precondition` shows "Videos are off on the server" alone**, not the sign-in notice
  `failureOf` would give it.
- **The `PUT` needs the bucket host in `connect-src`** (`public/_headers`, see [Deploy](deploy.md)), and the bucket's
  CORS rules must allow a `PUT` with `Content-Type` from `https://backoffice.clickplanet.lol`.
- **`npm run demo`**: three videos whose files are not in the demo (the page says it cannot play them), and a made-up
  bucket that takes a file in two seconds (`DemoAdmin.receive`, which the demo's `putFile` calls). A video uploaded
  there plays from the page's memory until a reload.
