uniform sampler2D atlasTexture;
uniform vec2 atlasTextureSize;
uniform float flagPaint;
uniform float pixelRatio;
uniform float shieldMost;
uniform sampler2D shieldMarks;
uniform vec2 shieldCells;
uniform float time;

flat in vec4 vRegionVector;
flat in float vSpriteSize;
flat in float vShield;
flat in float vPulse;
flat in float vHighlit;

varying float vHover;
varying float vGlow;
varying float vScorch;
varying float vFortify;

const float HIGHLIGHT_LIFT = 0.2;
const float HIGHLIGHT_FLOOR = 0.35;

// Must match MISSION_COLOUR in borderLines.ts and --mission in tokens.css.
const vec3 MISSION = vec3(0.784, 0.341, 0.298);
const float MISSION_RING_PX = 2.5;

// A capital still standing: a crown on the top of the tile, drawn last, over its flag, its shield and every ring.
const vec2 CROWN_AT = vec2(0.0, -0.34);
const float CROWN_SIZE = 0.4;

bool marked(float bit) {
    return mod(floor(vPulse / bit), 2.0) > 0.5;
}

#include ../flagFrame.glsl;
#include ../effectColours.glsl;

vec2 atlasUVof(vec4 region, vec2 uv) {
    return frameUVof(atlasFrameOf(region), uv);
}

vec4 ownColour() {
    if (vRegionVector.z == 0.0 || vRegionVector.w == 0.0) {
        return vec4(1.0, 1.0, 1.0, vHover > 0.5 ? 0.6 : 0.3);
    }

    float regionAspect = vRegionVector.z / vRegionVector.w;
    vec2 uv = gl_PointCoord;
    if (regionAspect > 1.0) {
        uv.x = (uv.x - 0.5) / regionAspect + 0.5;
    } else {
        uv.y = (uv.y - 0.5) * regionAspect + 0.5;
    }
    uv.y = 1.0 - uv.y;

    vec4 colour = texture2D(atlasTexture, atlasUVof(vRegionVector, uv));
    colour.a = vHover > 0.5 ? 1.0 : 0.7;
    return colour;
}

vec4 shieldMark(float cell, vec2 coordinates) {
    vec2 at = vec2(mod(cell, shieldCells.x), floor(cell / shieldCells.x)) + coordinates + 0.5;
    return texture(shieldMarks, vec2(at.x / shieldCells.x, 1.0 - at.y / shieldCells.y));
}

vec4 shielded(vec4 colour, vec2 coordinates) {
    float count = clamp(floor(vShield + 0.5), 1.0, shieldMost);
    float counted = smoothstep(16.0, 22.0, vSpriteSize / pixelRatio);
    vec4 mark = mix(shieldMark(0.0, coordinates), shieldMark(count, coordinates), counted);

    float cover = mark.a * (1.0 - flagPaint);
    return vec4(mix(colour.rgb, mark.rgb, cover), mix(colour.a, 1.0, cover));
}

vec4 crowned(vec4 colour, vec2 coordinates, float shown) {
    vec2 at = (coordinates - CROWN_AT) / CROWN_SIZE;
    if (max(abs(at.x), abs(at.y)) > 0.5) return colour;
    vec4 mark = shieldMark(shieldMost + 1.0, at);
    float cover = mark.a * shown;
    return vec4(mix(colour.rgb, mark.rgb, cover), mix(colour.a, 1.0, cover));
}

void main() {
    vec2 coordinates = gl_PointCoord - vec2(0.5);
    if (length(coordinates) > 0.5) discard;

    // The territory layer under the tiles paints the flags and the effects as the tiles fade out.
    float shown = 1.0 - flagPaint;
    vec4 colour = shown > 0.0 ? ownColour() : vec4(0.0);
    colour.a *= shown;

    if (vHighlit > 0.5) colour = vec4(mix(colour.rgb, vec3(1.0), HIGHLIGHT_LIFT), max(colour.a, HIGHLIGHT_FLOOR * shown));

    if (vShield > 0.5) colour = shielded(colour, coordinates);

    colour = scorched(colour, vScorch * shown);

    float aimed = marked(4.0) ? shown * smoothstep(10.0, 16.0, vSpriteSize / pixelRatio) : 0.0;
    if (aimed > 0.0) {
        float pixel = 1.0 / vSpriteSize;
        float inner = 0.5 - pixel - max(MISSION_RING_PX * pixelRatio * pixel, 0.06);
        float r = length(coordinates);
        float ring = smoothstep(inner - pixel, inner + pixel, r) * (1.0 - smoothstep(0.5 - 2.0 * pixel, 0.5, r));
        colour.rgb = mix(colour.rgb, MISSION, ring * aimed);
        colour.a = max(colour.a, ring * aimed);
    }

    float rings = mod(vPulse, 4.0);
    if (rings > 0.5) {
        float beat = 0.5 + 0.5 * sin(time * 6.2831853 / 1.1);
        float r = length(coordinates);
        float ring = smoothstep(0.31, 0.38, r) * (1.0 - smoothstep(0.46, 0.5, r));
        vec3 gold = vec3(1.0, 0.8, 0.22);
        vec3 tint = mod(rings, 2.0) > 0.5 ? gold : vec3(1.0);
        colour.rgb = mix(colour.rgb, tint, ring * (0.5 + 0.5 * beat));
        colour.a = max(colour.a, ring * (0.6 + 0.4 * beat));
    }

    colour = steeled(colour, vFortify * shown);
    colour = heated(colour, vGlow * shown);

    float capital = marked(8.0) ? shown * smoothstep(10.0, 16.0, vSpriteSize / pixelRatio) : 0.0;
    if (capital > 0.0) colour = crowned(colour, coordinates, capital);

    gl_FragColor = colour;
}
