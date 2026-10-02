#ifdef LIT
#include ../light.glsl;
#endif

uniform sampler2D atlasTexture;
uniform vec2 atlasTextureSize;
uniform float flagPaint;
uniform float pixelRatio;

flat in vec4 vRegionVector;
flat in vec2 vFlagUV;
flat in vec4 vFlagStep;
flat in vec4 vFlagRegion;
flat in float vFlagShare;
flat in float vSpriteSize;
#ifdef LIT
flat in float vShade;
flat in float vHaze;
#endif

varying float vHover;
varying float vGlow;
varying float vScorch;

const float KEYLINE_WIDTH = 1.5;
const float KEYLINE_FEATHER = 0.5;
const float KEYLINE_MOST = 0.03;

vec2 atlasUVof(vec4 region, vec2 uv) {
    vec2 origin = vec2(region.x, atlasTextureSize.y - region.y - region.w) / atlasTextureSize;
    return origin + clamp(uv, 0.002, 0.998) * (region.zw / atlasTextureSize);
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

void main() {
    vec2 coordinates = gl_PointCoord - vec2(0.5);
    if (length(coordinates) > 0.5) discard;

    vec4 own = flagPaint < 1.0 ? ownColour() : vec4(0.0);

    vec4 painted = vec4(own.rgb, 0.0);
    if (vFlagRegion.z > 0.0) {
        // gl_PointCoord runs down and the flag frame runs up; the frame itself needs no flip.
        vec2 screen = vec2(coordinates.x, -coordinates.y) * vSpriteSize;
        vec2 uv = vFlagUV + vFlagStep.xy * screen.x + vFlagStep.zw * screen.y;

        // Explicit gradients: the frame is flat across the sprite, so texture() picks the top mip.
        vec2 footprint = vFlagRegion.zw / atlasTextureSize;
        vec3 flag = textureGrad(
            atlasTexture, atlasUVof(vFlagRegion, uv),
            vFlagStep.xy * footprint, vFlagStep.zw * footprint
        ).rgb;

        float uvPerPixel = max(length(vFlagStep.xy), length(vFlagStep.zw));
        float width = min(uvPerPixel * KEYLINE_WIDTH * pixelRatio, KEYLINE_MOST);
        float feather = min(uvPerPixel * KEYLINE_FEATHER, width * 0.5);
        float inner = 0.5 - width;
        float edge = max(abs(uv.x - 0.5), abs(uv.y - 0.5));
        float keyline = smoothstep(inner - feather, inner + feather, edge)
            * (1.0 - smoothstep(0.5 - feather, 0.5 + feather, edge));
        flag = mix(flag, vec3(0.03), keyline * 0.85 * vFlagShare);

        painted = vec4(flag, vFlagShare * (vHover > 0.5 ? 1.0 : 0.94));
    }

    vec4 colour = mix(own, painted, flagPaint);

#ifdef LIT
    colour.rgb = lit(colour.rgb, vShade, vHaze);
#endif

    vec3 ember = mix(vec3(0.07, 0.02, 0.01), vec3(1.0, 0.32, 0.04), vScorch * vScorch);
    colour = mix(colour, vec4(ember, 0.95), vScorch * 0.9);

    vec3 hot = mix(vec3(1.0, 0.42, 0.08), vec3(1.0, 0.96, 0.82), vGlow * vGlow);
    colour = mix(colour, vec4(hot, 1.0), vGlow);

    gl_FragColor = colour;
}
