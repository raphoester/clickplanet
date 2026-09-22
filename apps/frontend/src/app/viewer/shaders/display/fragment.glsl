uniform sampler2D atlasTexture;
uniform vec2 atlasTextureSize;
uniform float flagPaint;

flat in vec4 vRegionVector;
flat in vec2 vFlagUV;
flat in vec4 vFlagStep;
flat in vec4 vFlagRegion;
flat in float vFlagShare;
flat in float vSpriteSize;

varying float vHover;
varying float vGlow;
varying float vScorch;

// How wide the keyline around a painted flag is drawn, in screen pixels, how
// much of a pixel its edges are softened over, and the most of the flag's own
// half-width it may ever take.
const float KEYLINE_WIDTH = 1.5;
const float KEYLINE_FEATHER = 0.5;
const float KEYLINE_MOST = 0.03;

vec2 atlasUVof(vec4 region, vec2 uv) {
    vec2 origin = vec2(region.x, atlasTextureSize.y - region.y - region.w) / atlasTextureSize;
    return origin + clamp(uv, 0.002, 0.998) * (region.zw / atlasTextureSize);
}

// The tile on its own: its owner's flag, drawn into the disc. `flagPaint` is
// what keeps it off the screen until a tile is big enough to read one — below
// that size a 100px flag in a 2px disc is noise either way, mip-blurred to grey
// or aliased into sparkle.
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

    // Zoomed out the tile's own flag is mixed all the way out again below, so
    // the fetch it costs buys nothing there.
    vec4 own = flagPaint < 1.0 ? ownColour() : vec4(0.0);

    vec4 painted = vec4(own.rgb, 0.0);
    if (vFlagRegion.z > 0.0) {
        // Where *this fragment* falls in the flag, not where its tile does.
        // The flag is laid on the ground, so reading it under the pixel is what
        // makes it one painted image: neighbouring discs overlap — they are
        // widened until they cover the ground, or the flag would never reach it
        // — and a per-tile sample had each of them paint its own flat colour,
        // so the flag came out at the resolution of the tile lattice rather
        // than of the screen. On a landmass a handful of tiles across that is
        // a few samples of a 100px flag: bands survive it, a coat of arms or a
        // sun does not.
        //
        // `gl_PointCoord` runs down the sprite and the frame runs up, hence the
        // sign; the flag frame itself needs no turning over, because it and the
        // atlas are both built with 0 at the bottom. Turning it over here, as
        // the per-tile path has to, is what had every painted flag upside down.
        vec2 screen = vec2(coordinates.x, -coordinates.y) * vSpriteSize;
        vec2 uv = vFlagUV + vFlagStep.xy * screen.x + vFlagStep.zw * screen.y;

        // Sampled over the footprint one pixel actually covers. Without the
        // gradients the driver has none to work from — the frame is flat across
        // the sprite — and takes the top mip, which point-samples a 100px flag
        // squeezed into a few dozen pixels and turns every device on it into
        // sparkle.
        vec2 footprint = vFlagRegion.zw / atlasTextureSize;
        vec3 flag = textureGrad(
            atlasTexture, atlasUVof(vFlagRegion, uv),
            vFlagStep.xy * footprint, vFlagStep.zw * footprint
        ).rgb;

        // A keyline around the flag's own rectangle, so its stripes do not read
        // as more territory. It fades with the flag it belongs to.
        // Only where the flag itself ends, never out in the extended colour.
        //
        // Held to a width in pixels, like the countries' own outline: as a
        // share of the flag it is a sliver on a small landmass and a bar on a
        // big one, and now that the edge is a real line rather than a ring of
        // discs, that difference shows. Capped as a share of the flag all the
        // same, for the faintest ones, where a pixel is a good part of the
        // whole rectangle and a line drawn in pixels would eat it.
        float uvPerPixel = max(length(vFlagStep.xy), length(vFlagStep.zw));
        float width = min(uvPerPixel * KEYLINE_WIDTH, KEYLINE_MOST);
        float feather = min(uvPerPixel * KEYLINE_FEATHER, width * 0.5);
        float inner = 0.5 - width;
        float edge = max(abs(uv.x - 0.5), abs(uv.y - 0.5));
        float keyline = smoothstep(inner - feather, inner + feather, edge)
            * (1.0 - smoothstep(0.5 - feather, 0.5 + feather, edge));
        flag = mix(flag, vec3(0.03), keyline * 0.85 * vFlagShare);

        // Opacity is the leader's share of this piece of land, so ground nobody
        // has settled stays the Earth underneath.
        painted = vec4(flag, vFlagShare * (vHover > 0.5 ? 1.0 : 0.94));
    }

    vec4 colour = mix(own, painted, flagPaint);

    // Bombs, laid over both the tile and the painted flag so they read at any
    // zoom. The crater glows like embers and cools to char.
    vec3 ember = mix(vec3(0.07, 0.02, 0.01), vec3(1.0, 0.32, 0.04), vScorch * vScorch);
    colour = mix(colour, vec4(ember, 0.95), vScorch * 0.9);

    vec3 hot = mix(vec3(1.0, 0.42, 0.08), vec3(1.0, 0.96, 0.82), vGlow * vGlow);
    colour = mix(colour, vec4(hot, 1.0), vGlow);

    gl_FragColor = colour;
}
