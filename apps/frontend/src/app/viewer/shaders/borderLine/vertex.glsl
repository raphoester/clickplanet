// One instance per cell edge, drawn as a quad that is built in *pixels*: the
// two ends are projected first and the quad is then laid out across the screen,
// so the line keeps the same width at every zoom instead of thinning to nothing
// as the globe is pushed away.

// Half the drawing buffer, in pixels: what turns a pixel offset into clip space.
uniform vec2 halfViewport;

// Half the line's width in pixels, feather included.
uniform float halfWidth;

// The shell the line is drawn on. Just outside the tiles it runs over them;
// just inside, the tiles' own depth hides whatever they cover.
uniform float lift;

attribute vec2 corner;
attribute vec3 from;
attribute vec3 to;

// How far across the line this fragment is, in pixels. The edge is softened
// from it rather than from the geometry, so a line under two pixels wide is
// still a line and not a row of gaps.
out float vAcross;

void main() {
    vec4 head = projectionMatrix * modelViewMatrix * vec4(normalize(from) * lift, 1.0);
    vec4 tail = projectionMatrix * modelViewMatrix * vec4(normalize(to) * lift, 1.0);

    vec2 span = tail.xy / tail.w * halfViewport - head.xy / head.w * halfViewport;
    float length2 = length(span);
    // A cell edge is never zero-length, but it projects to nothing when it
    // points straight at the camera at the globe's limb.
    vec2 along = length2 > 1e-6 ? span / length2 : vec2(1.0, 0.0);

    // The quad runs from one end to the other and out past both, so that the
    // next edge's quad meets this one at the corner they share.
    vec4 clip = corner.x < 0.0 ? head : tail;
    vec2 offset = vec2(-along.y, along.x) * corner.y * halfWidth + along * corner.x * halfWidth;
    clip.xy += offset / halfViewport * clip.w;

    vAcross = corner.y * halfWidth;
    gl_Position = clip;
}
