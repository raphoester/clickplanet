#include ../light.glsl;

uniform sampler2D map;

varying vec2 vUv;
varying vec3 vViewNormal;

const vec3 VIEW_DIRECTION = vec3(0.0, 0.0, 1.0);

const float GLINT_SHARPNESS = 60.0;
const float GLINT = 0.35;
const float SHEEN_SHARPNESS = 16.0;
const float SHEEN = 0.06;
const vec3 GLINT_COLOUR = vec3(1.0, 0.97, 0.9);

float waterOf(vec3 texel) {
    return smoothstep(0.12, 0.28, texel.b - max(texel.r, texel.g));
}

void main() {
    vec3 normal = normalize(vViewNormal);
    vec3 texel = texture2D(map, vUv).rgb;

    vec3 colour = texel;

    float towards = max(dot(normal, normalize(normalize(LIGHT_FROM) + VIEW_DIRECTION)), 0.0);
    float glint = GLINT * pow(towards, GLINT_SHARPNESS) + SHEEN * pow(towards, SHEEN_SHARPNESS);
    colour += GLINT_COLOUR * glint * waterOf(texel);

    gl_FragColor = vec4(colour, 1.0);
}
