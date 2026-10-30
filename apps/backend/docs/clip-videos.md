# Clip videos (`internal/clips/`, in the server)

**The videos the operator posts by hand**, such as the season-end timelapses. They are uploaded on the backoffice's
Videos page from a computer, then downloaded on a phone, where the TikTok app picks the sound (no API sets a TikTok
library sound). Kept by the clips module, beside its stats and casts, because they are the clips' videos (the user,
2026-10-30). Shown on the backoffice's [Videos](../../backoffice/docs/videos.md) page.

```
internal/clips/internal/
  videos/                             VideoID (VideoIDOf), File, MediaType, Rules (FileError), Video (Keep, TakeSnapshot),
                                      VideoSnapshot (Restore), FileName; the IDProvider port; fakes behind the testing tag
    s3_video_bucket/                  the files, in the bucket under clips/videos/<video_id>: upload, view and download links,
                                      what a file uploaded is, and its deletion
    postgres_video_store/             Keep (a video kept again takes its new name and caption) and Remove, over clips.videos
    uuid_id_provider/                 a random UUID per video
    usecases/create_upload_usecase/   draws an id, checks the type and the size, signs the upload link
    usecases/keep_video_usecase/      reads the file back from the bucket, keeps the video with what the bucket holds
    usecases/remove_video_usecase/    deletes the file, then the row
  clipsv1controller/
    create_video_upload_handler/  keep_video_handler/  remove_video_handler/
    list_videos_handler/videos_query/   PostgresQuery: every video, the newest first (200), each with its two links
```

**`clips.v1.AdminService`, on the admin listener**: `CreateVideoUpload`, `KeepVideo`, `ListVideos`, `RemoveVideo`.

- **The file first, the row after**, as a support note's files: `CreateVideoUpload(content_type, size_bytes)` answers
  an id and a link, and the page sends the file there with one `PUT`, straight to the bucket. No byte passes through
  the server (1 GB of memory, and Cloudflare cuts a proxied body at 100 MB). Then `KeepVideo(video_id, name, caption)`
  reads the file back with a `HEAD` and keeps the type and the size the bucket holds, never what the page said. A file
  that is not there is `FailedPrecondition`.
- **An mp4, a mov or a webm, of 500 MiB at most** (`clips.videos.maxBytes`): anything else is `InvalidArgument`, at the
  link and again at the keep. A name of 1 to 100 characters, a caption of 2,200 at most (TikTok's limit).
- **Two links per video, good for `linkTTL` (6h)**: a view link the page plays, and a download link signed with
  `response-content-disposition: attachment`, so a phone saves the file under its name (`videos.FileName`: letters,
  digits, `-` and `_`, and the type's extension) instead of playing it. A page left open longer reads the list again.
- **A removal deletes the file, then the row.** A removal that fails halfway leaves the video listed, and asked again
  it is removed: a deletion of a file the bucket no longer holds is a success.
- **An upload never kept leaves its file in the bucket.** Only an operator uploads, a few files a season, so nothing
  sweeps them. Nothing counts these files against the support module's `maxStoredBytes` either.
- **Off** (`clips.videos.enabled: false`), every video call is `FailedPrecondition`, and the bucket block is not read.
- **Its table** (`clips` schema, migration `20261030120000_videos`): `videos(video_id, name, caption, media_type,
  size_bytes, kept_at)`.
- **One bucket for both modules**, `clickplanet-media` in production, through `cps3` (see [Shared](shared.md)), with
  the same key: support's files at its root, the videos under `clips/videos/`. Its CORS lets
  `https://backoffice.clickplanet.lol` `PUT`, and the backoffice's CSP names the bucket's host in `connect-src` (see
  `deploy/vps/README.md`, "Photos and videos on a note").
