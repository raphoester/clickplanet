// A slime's quad, laid on the ground between its source and its target and wide enough for its droplets. Lengths in
// the fragment shader are in tile spacings, x along the way from the source to the target.
uniform float spacing;
uniform float grow;
uniform float lift;

attribute vec3 from;
attribute vec3 to;
// along, head, source, neck
attribute vec4 shape;
// drops, drop, wobble, spray
attribute vec4 splash;
// alpha, dead, gloss, flash
attribute vec4 tone;
attribute float seed;

varying vec2 vLocal;
flat out float vLength;
flat out vec4 vShape;
flat out vec4 vSplash;
flat out vec4 vTone;
flat out float vSeed;
flat out vec3 vLight;
flat out vec3 vView;

// From the viewer's upper left, whichever way the goo lies on the globe.
const vec3 LIGHT = vec3(-0.4588, 0.6117, 0.6443);

void main() {
    vec3 a = normalize(from);
    vec3 b = normalize(to);
    vec3 mid = normalize(a + b);
    vec3 axis = (b - a) - dot(b - a, mid) * mid;
    float span = length(axis);
    if (span < 1e-7) {
        axis = cross(vec3(0.0, 1.0, 0.0), mid);
        if (length(axis) < 1e-6) axis = vec3(1.0, 0.0, 0.0);
    }
    axis = normalize(axis);
    vec3 side = cross(mid, axis);

    if (tone.x <= 0.0 || (normalMatrix * mid).z < 0.0) {
        gl_Position = vec4(2.0, 2.0, 2.0, 1.0);
        return;
    }

    float run = span / spacing;
    float reach = grow * (max(shape.y, shape.z) * 1.35 + splash.x + splash.y * 1.5 + 0.05);
    vec2 local = position.xy * vec2(run + 2.0 * reach, 2.0 * reach);
    vec3 ground = mid + (axis * local.x + side * local.y) * spacing;

    // Over the infected tiles, which are drawn lift nearer the camera; the camera is orthographic, so only the depth
    // moves.
    vec4 seen = modelViewMatrix * vec4(ground, 1.0);
    seen.z += 2.0 * lift;
    gl_Position = projectionMatrix * seen;

    vLocal = local;
    vLength = run;
    vShape = shape;
    vSplash = splash;
    vTone = tone;
    vSeed = seed;

    // The light and the viewer in the quad's own frame: x along, y across, z out of the ground.
    mat3 toView = mat3(modelViewMatrix);
    vec3 light = LIGHT * toView;
    vec3 viewer = vec3(0.0, 0.0, 1.0) * toView;
    vLight = vec3(dot(light, axis), dot(light, side), dot(light, mid));
    vView = vec3(dot(viewer, axis), dot(viewer, side), dot(viewer, mid));
}
