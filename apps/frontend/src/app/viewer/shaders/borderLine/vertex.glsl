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

// How far past the globe's limb this pass can still be seen, as the sine of
// that angle: the Earth is opaque at 0.999, so a line on `lift` shows out to
// acos(0.999 / lift) and is covered by the Earth's silhouette past it. Worked
// out on the CPU, where `lift` is decided.
uniform float limb;

attribute vec2 corner;
attribute vec3 from;
attribute vec3 to;

// How far across the line this fragment is, in pixels. The edge is softened
// from it rather than from the geometry, so a line under two pixels wide is
// still a line and not a row of gaps.
out float vAcross;

void main() {
    vec3 headDirection = normalize(from);
    vec3 tailDirection = normalize(to);

    // A piece with both ends behind the globe is hidden by the opaque Earth,
    // and half the outline is behind the globe at any moment. Dropping it here
    // costs two dot products and saves it being laid out across the screen
    // only to be thrown away by the depth test.
    //
    // The camera is orthographic, so one world unit is `halfViewport.y * zoom`
    // pixels and `projectionMatrix[1][1]` is that zoom: the line's own width
    // is what the limb has to be widened by, so a piece that still shows a
    // sliver is kept.
    float slack = limb + (halfWidth + 1.0) / max(halfViewport.y * projectionMatrix[1][1], 1.0);
    if ((normalMatrix * headDirection).z < -slack && (normalMatrix * tailDirection).z < -slack) {
        gl_Position = vec4(2.0, 2.0, 2.0, 1.0);
        return;
    }

    vec4 head = projectionMatrix * modelViewMatrix * vec4(headDirection * lift, 1.0);
    vec4 tail = projectionMatrix * modelViewMatrix * vec4(tailDirection * lift, 1.0);

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
