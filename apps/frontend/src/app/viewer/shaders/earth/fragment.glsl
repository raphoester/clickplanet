#ifdef SEA_GLINT
#include ../light.glsl;
#endif

uniform sampler2D map;

varying vec2 vUv;
varying vec3 vViewNormal;

// The photo as it is is too deep a blue: a little less colour, and its darks lifted.
const float SATURATION = 0.75;
const float LIFT = 1.25;

#ifdef SEA_GLINT
const vec3 VIEW_DIRECTION = vec3(0.0, 0.0, 1.0);

const float GLINT_SHARPNESS = 60.0;
const float GLINT = 0.35;
const float SHEEN_SHARPNESS = 16.0;
const float SHEEN = 0.06;
const vec3 GLINT_COLOUR = vec3(1.0, 0.97, 0.9);

float waterOf(vec3 texel) {
    return smoothstep(0.12, 0.28, texel.b - max(texel.r, texel.g));
}
#endif

vec3 softened(vec3 texel) {
    float grey = dot(texel, vec3(0.2126, 0.7152, 0.0722));
    return pow(mix(vec3(grey), texel, SATURATION), vec3(1.0 / LIFT));
}

void main() {
    vec3 texel = texture2D(map, vUv).rgb;
    vec3 colour = softened(texel);

#ifdef SEA_GLINT
    vec3 normal = normalize(vViewNormal);
    float towards = max(dot(normal, normalize(normalize(LIGHT_FROM) + VIEW_DIRECTION)), 0.0);
    float glint = GLINT * pow(towards, GLINT_SHARPNESS) + SHEEN * pow(towards, SHEEN_SHARPNESS);
    colour += GLINT_COLOUR * glint * waterOf(texel);
#endif

    gl_FragColor = vec4(colour, 1.0);
}
