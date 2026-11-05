package posts

// What a clip was, as the generator wrote it: a `clips.v1.ClipRecord` in protojson, kept with its post and never
// published, so a later step can set it beside how the post did. The job neither reads nor chooses from it; empty for a
// clip written before the generator kept one.
type ClipRecord []byte

// The teaser of a clip written before its teasers were named.
const unnamedTeaser = "unnamed"
