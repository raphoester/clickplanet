uniform sampler2D atlasTexture;
uniform vec2 atlasTextureSize;

flat in vec4 vRegionVector;
varying float vHover;

vec4 applyHover(vec4 color, float hover) {
    color.a = 0.7;
    if (hover > 0.5) {
        color.a = 1.0;
    }

    return color;
}

void main() {
    vec2 coordinates = gl_PointCoord - vec2(0.5);
    float dist = length(coordinates);
    if (dist > 0.5) discard;

    if (vRegionVector.z == 0.0 || vRegionVector.w == 0.0) {
        gl_FragColor = vec4(1.0, 1.0, 1.0, 0.3);
        if (vHover > 0.5) {
            gl_FragColor.a = 0.6;
        }
        return;
    }

    vec2 uv = gl_PointCoord;

    vec2 normalizedRegionXY = vec2(vRegionVector.x, atlasTextureSize.y - vRegionVector.y - vRegionVector.w) / atlasTextureSize;
    vec2 normalizedRegionZW = vRegionVector.zw / atlasTextureSize;

    float regionAspect = normalizedRegionZW.x / normalizedRegionZW.y;
    float shapeAspect = 1.0;

    vec2 adjustedUV = vec2(0.0);
    if (regionAspect > shapeAspect) {
        float scale = shapeAspect / regionAspect;
        adjustedUV.x = (uv.x - 0.5) * scale + 0.5;
        adjustedUV.y = uv.y;
    } else {
        float scale = regionAspect / shapeAspect;
        adjustedUV.x = uv.x;
        adjustedUV.y = (uv.y - 0.5) * scale + 0.5;
    }

    adjustedUV.y = 1.0 - adjustedUV.y;

    vec2 atlasUV = normalizedRegionXY + adjustedUV * normalizedRegionZW;

    gl_FragColor = texture2D(atlasTexture, atlasUV);
    gl_FragColor = applyHover(gl_FragColor, vHover);
}
