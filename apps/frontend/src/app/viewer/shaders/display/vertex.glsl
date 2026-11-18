uniform float pointSize;

uniform float pixelsPerRadian;

// The provinces lit, first and last: a landmass's span, or one province.
uniform vec2 highlight;

uniform float pixelRatio;

#include ../tileEffects.glsl;

// A tile a flag close to fortifying still misses; the fragment shader and globe.ts's BEAT_SECONDS beat at the same rate.
const float PULSE_RATE = 6.2831853 / 1.1;
uniform float pulseTime;
// An infected tile's sprite holds its pods past its flag, and is drawn virusLift nearer the camera, so its pods lie over
// the tiles next to it. More than the depth between two neighbours, less than a tile at the limb behind the earth: in the
// board's own spacings (globe.ts).
const float VIRUS_REACH = 1.8;
uniform float virusLift;

varying float vGlow;
varying float vScorch;
varying float vFortify;

#ifdef COMPACT
uniform sampler2D flagRectangles;

attribute vec2 ids;
attribute vec4 marks;

#define regionVector texelFetch(flagRectangles, ivec2(int(ids.x), 0), 0)
#define provinceIndex ids.y
#define hover marks.x
#define pulse marks.y
#define shield marks.z
#else
// Must match PACK in tileField.ts: the flag's atlas rectangle as x + y * PACK and width + height * PACK.
const float PACK = 2048.0;

attribute vec2 flagRegion;
attribute vec4 tileMarks;

#define regionVector vec4(mod(flagRegion, PACK), floor(flagRegion / PACK)).xzyw
#define provinceIndex tileMarks.x
#define hover tileMarks.y
#define pulse tileMarks.z
#define shield tileMarks.w
#endif

varying float vHover;
flat out vec4 vRegionVector;
flat out float vShield;
flat out float vPulse;
flat out float vHighlit;
// A turn of its own for each tile's marks, from where it is.
flat out float vSeed;
flat out float vReach;

flat out float vSpriteSize;

// Must match LIMB and THROW in tileField.ts, which skips the pieces of the map this drops whole.
const float LIMB = 0.05;

void main() {
    vec3 ground = normalize(position);

    float thrown = 0.0;
    for (int i = 0; i < MAX_BLASTS; i++) thrown = max(thrown, blastRadii[i]);
    float limb = LIMB + (pointSize * 0.5 + 1.0) / max(pixelsPerRadian, 1.0) + motion * thrown * 0.6;

    if ((normalMatrix * ground).z < -limb) {
        gl_Position = vec4(2.0, 2.0, 2.0, 1.0);
        gl_PointSize = 0.0;
        vSpriteSize = 0.0;
        return;
    }

    vHover = hover;
    vRegionVector = regionVector;
    vShield = shield;
    vPulse = pulse;
    vHighlit = provinceIndex > highlight.x - 0.5 && provinceIndex < highlight.y + 0.5 ? 1.0 : 0.0;
    vSeed = fract(sin(dot(ground, vec3(12.9898, 78.233, 37.719))) * 43758.5453) * 6.2831853;

    TileEffects effects = tileEffectsAt(ground, provinceIndex);
    vGlow = effects.glow;
    vScorch = effects.scorch;
    vFortify = effects.fortify;

    float swell = effects.swell;
    if (mod(pulse, 4.0) > 0.5) swell = max(swell, 0.12 * (0.5 + 0.5 * sin(pulseTime * PULSE_RATE)));

    float tileSize = pointSize * (1.0 + motion * swell);
    bool infected = mod(floor(pulse / 16.0), 2.0) > 0.5;
    vReach = infected ? mix(1.0, VIRUS_REACH, smoothstep(9.0, 15.0, tileSize / pixelRatio)) : 1.0;
    vSpriteSize = tileSize * vReach;
    gl_PointSize = vSpriteSize;
    vec4 seen = modelViewMatrix * vec4(position + effects.push, 1.0);
    if (infected) seen.z += virusLift;
    gl_Position = projectionMatrix * seen;
}
