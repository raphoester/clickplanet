#ifdef SEA_GLINT
#include ../light.glsl;
#endif

uniform vec3 sea;

varying vec3 vViewNormal;

#ifdef SEA_GLINT
const vec3 VIEW_DIRECTION = vec3(0.0, 0.0, 1.0);

const float GLINT_SHARPNESS = 60.0;
const float GLINT = 0.35;
const float SHEEN_SHARPNESS = 16.0;
const float SHEEN = 0.06;
const vec3 GLINT_COLOUR = vec3(1.0, 0.97, 0.9);
#endif

void main() {
    vec3 colour = sea;

#ifdef SEA_GLINT
    vec3 normal = normalize(vViewNormal);
    float towards = max(dot(normal, normalize(normalize(LIGHT_FROM) + VIEW_DIRECTION)), 0.0);
    colour += GLINT_COLOUR * (GLINT * pow(towards, GLINT_SHARPNESS) + SHEEN * pow(towards, SHEEN_SHARPNESS));
#endif

    gl_FragColor = vec4(colour, 1.0);
}
