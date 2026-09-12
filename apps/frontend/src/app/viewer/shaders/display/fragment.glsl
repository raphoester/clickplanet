uniform sampler2D atlasTexture;
uniform vec2 atlasTextureSize;
uniform float flagPaint;

flat in vec4 vRegionVector;
flat in vec2 vFlagUV;
flat in vec4 vFlagRegion;
flat in float vFlagShare;
varying float vHover;
varying float vGlow;
varying float vScorch;

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

    vec4 own = ownColour();

    vec4 painted = vec4(own.rgb, 0.0);
    if (vFlagRegion.z > 0.0) {
        // `vFlagUV` already runs the way `atlasUVof` wants it — both are 0 at
        // the bottom, the atlas because the texture is uploaded flipped and the
        // frame because it is built on the landmass's north axis. Turning it
        // over here, as the per-tile path has to for `gl_PointCoord`, is what
        // had every painted flag upside down.
        vec3 flag = texture(atlasTexture, atlasUVof(vFlagRegion, vFlagUV)).rgb;

        // A keyline around the flag's own rectangle, so its stripes do not read
        // as more territory. It fades with the flag it belongs to.
        // Only where the flag itself ends, never out in the extended colour.
        float edge = max(abs(vFlagUV.x - 0.5), abs(vFlagUV.y - 0.5));
        float keyline = step(0.478, edge) * step(edge, 0.5);
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
