vec2 frameUVof(vec4 frame, vec2 uv) {
    return frame.xy + clamp(uv, 0.002, 0.998) * frame.zw;
}

vec4 atlasFrameOf(vec4 region) {
    return vec4(region.x, atlasTextureSize.y - region.y - region.w, region.zw) / atlasTextureSize.xyxy;
}

// A flag's empty part is white on the globe: Nepal's corners, the one flag that is not a rectangle. Its colour there,
// under an alpha of 0, is whatever the file kept (blue streaks in the atlas) or black where the browser drops it.
vec3 onWhite(vec4 flag) {
    return mix(vec3(1.0), flag.rgb, flag.a);
}
