vec2 frameUVof(vec4 frame, vec2 uv) {
    return frame.xy + clamp(uv, 0.002, 0.998) * frame.zw;
}

vec4 atlasFrameOf(vec4 region) {
    return vec4(region.x, atlasTextureSize.y - region.y - region.w, region.zw) / atlasTextureSize.xyxy;
}
