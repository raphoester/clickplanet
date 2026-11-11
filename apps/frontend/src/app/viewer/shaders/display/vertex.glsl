uniform float pointSize;

uniform float pixelsPerRadian;

uniform float highlight;

#include ../tileEffects.glsl;

// A tile a flag close to fortifying still misses; the fragment shader beats at the same rate.
const float PULSE_RATE = 6.2831853 / 1.1;

varying float vGlow;
varying float vScorch;
varying float vFortify;

#ifdef COMPACT
uniform sampler2D flagRectangles;

attribute vec2 ids;
attribute vec4 marks;

#define regionVector texelFetch(flagRectangles, ivec2(int(ids.x), 0), 0)
#define landmassIndex ids.y
#define hover marks.x
#define pulse marks.y
#define shield marks.z
#else
attribute float hover;
attribute vec4 regionVector;
attribute float landmassIndex;
attribute float shield;
attribute float pulse;
#endif

varying float vHover;
flat out vec4 vRegionVector;
flat out float vShield;
flat out float vPulse;
flat out float vHighlit;

flat out float vSpriteSize;

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
    vHighlit = highlight > 0.5 && abs(landmassIndex - highlight) < 0.5 ? 1.0 : 0.0;

    TileEffects effects = tileEffectsAt(ground, landmassIndex);
    vGlow = effects.glow;
    vScorch = effects.scorch;
    vFortify = effects.fortify;

    float swell = effects.swell;
    if (mod(pulse, 4.0) > 0.5) swell = max(swell, 0.12 * (0.5 + 0.5 * sin(time * PULSE_RATE)));

    vSpriteSize = pointSize * (1.0 + motion * swell);
    gl_PointSize = vSpriteSize;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position + effects.push, 1.0);
}
