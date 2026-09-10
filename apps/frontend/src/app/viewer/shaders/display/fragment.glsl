uniform sampler2D atlasTexture;
uniform vec2 atlasTextureSize;
uniform float flagDetail;
uniform float flagPaint;

flat in vec4 vRegionVector;
flat in vec2 vFlagUV;
flat in vec4 vFlagRegion;
flat in float vFlagShare;
varying float vHover;

vec2 atlasUVof(vec4 region, vec2 uv) {
    vec2 origin = vec2(region.x, atlasTextureSize.y - region.y - region.w) / atlasTextureSize;
    return origin + clamp(uv, 0.002, 0.998) * (region.zw / atlasTextureSize);
}

// The tile on its own: its owner's flag, drawn into the disc. Only ever seen
// once a tile is big enough to read, which is also the only zoom where the
// painted landmass flag has faded out.
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

    // Below the size a flag can be read at, the tile fades out rather than
    // sampling one arbitrary texel of it: a 100px flag in a 2px disc is noise
    // either way, mip-blurred to grey or aliased into sparkle.
    colour.a *= flagDetail;
    return colour;
}

void main() {
    vec2 coordinates = gl_PointCoord - vec2(0.5);
    if (length(coordinates) > 0.5) discard;

    vec4 own = ownColour();

    vec4 painted = vec4(own.rgb, 0.0);
    if (vFlagRegion.z > 0.0) {
        vec3 flag = texture(atlasTexture, atlasUVof(vFlagRegion, vec2(vFlagUV.x, 1.0 - vFlagUV.y))).rgb;

        // A keyline around the flag's own rectangle, so its stripes do not read
        // as more territory. It fades with the flag it belongs to.
        vec2 edge = abs(vFlagUV - 0.5);
        flag = mix(flag, vec3(0.03), step(0.478, max(edge.x, edge.y)) * 0.85 * vFlagShare);

        // Opacity is the leader's share of this piece of land, so ground nobody
        // has settled stays the Earth underneath.
        painted = vec4(flag, vFlagShare * (vHover > 0.5 ? 1.0 : 0.94));
    }

    gl_FragColor = mix(own, painted, flagPaint);
}
