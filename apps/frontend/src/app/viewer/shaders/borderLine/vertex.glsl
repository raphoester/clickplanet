uniform vec2 halfViewport;

uniform float halfWidth;

uniform float lift;

uniform float limb;

attribute vec2 corner;
attribute vec3 from;
attribute vec3 to;

#ifdef HIGHLIGHT
// The landmass or the province lit: a pair of the countries' pieces is of landmasses, of the provinces' lines of provinces.
uniform float highlit;
#ifdef PROVINCES
// The provinces either side of each of the countries' pieces.
attribute vec2 provinces;
#define litSides provinces
#else
attribute vec2 sides;
#define litSides sides
#endif
#endif

#ifdef MISSION
// Must match AIMED_COLUMNS in borderLines.ts.
#define AIMED_COLUMNS 1024
uniform sampler2D aimed;
attribute vec2 sides;

float aimedAt(float landmass) {
    int at = int(landmass + 0.5);
    return texelFetch(aimed, ivec2(at % AIMED_COLUMNS, at / AIMED_COLUMNS), 0).r;
}
#endif

out float vAcross;

#ifdef STACKED
// The most layers a piece inks as: past them, the line is solid.
#define MOST_LAYERS 64.0
// The cell edges a piece stands for, and their chord over their length.
attribute vec2 standsFor;
flat out float vLayers;
flat out float vLength;
out float vAlong;
#endif

void main() {
#ifdef HIGHLIGHT
    if (abs(litSides.x - highlit) > 0.5 && abs(litSides.y - highlit) > 0.5) {
        gl_Position = vec4(2.0, 2.0, 2.0, 1.0);
        return;
    }
#endif

#ifdef MISSION
    if (abs(aimedAt(sides.x) - aimedAt(sides.y)) < 0.5) {
        gl_Position = vec4(2.0, 2.0, 2.0, 1.0);
        return;
    }
#endif

    vec3 headDirection = normalize(from);
    vec3 tailDirection = normalize(to);

    // A coarse piece's cell edges slant across its line, and their quads laid over one another made the line wider.
#ifdef STACKED
    float across = halfWidth / standsFor.y;
#else
    float across = halfWidth;
#endif

    float slack = limb + (across + 1.0) / max(halfViewport.y * projectionMatrix[1][1], 1.0);
    if ((normalMatrix * headDirection).z < -slack && (normalMatrix * tailDirection).z < -slack) {
        gl_Position = vec4(2.0, 2.0, 2.0, 1.0);
        return;
    }

    vec4 head = projectionMatrix * modelViewMatrix * vec4(headDirection * lift, 1.0);
    vec4 tail = projectionMatrix * modelViewMatrix * vec4(tailDirection * lift, 1.0);

    vec2 span = tail.xy / tail.w * halfViewport - head.xy / head.w * halfViewport;
    float length2 = length(span);
    vec2 along = length2 > 1e-6 ? span / length2 : vec2(1.0, 0.0);

    vec4 clip = corner.x < 0.0 ? head : tail;
    vec2 offset = vec2(-along.y, along.x) * corner.y * across + along * corner.x * halfWidth;
    clip.xy += offset / halfViewport * clip.w;

    vAcross = corner.y * halfWidth;
#ifdef STACKED
    // How many of its cell edges' quads would have covered a pixel of this piece.
    vLayers = min(1.0 + 2.0 * halfWidth * standsFor.x * standsFor.y / max(length2, 1e-3), MOST_LAYERS);
    vLength = length2;
    vAlong = corner.x < 0.0 ? -halfWidth : length2 + halfWidth;
#endif
    gl_Position = clip;
}
