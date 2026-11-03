#ifdef SEA_GLINT
#include ../light.glsl;
#endif

uniform sampler2D map;

varying vec2 vUv;
varying vec3 vViewNormal;

// The photo as it is is too deep a blue: a little less colour, and its darks lifted.
const float SATURATION = 0.75;
const float LIFT = 1.25;

const vec3 LUMINANCE = vec3(0.2126, 0.7152, 0.0722);

// An unclaimed tile is white at 30%, so on white ground it vanishes: ice and snow are held to a light grey.
const float SHOULDER_FROM = 0.7;
const float BRIGHTEST = 0.82;
const float BEND = ((1.0 - SHOULDER_FROM) / (BRIGHTEST - SHOULDER_FROM) - 1.0) / (1.0 - SHOULDER_FROM);

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
    float grey = dot(texel, LUMINANCE);
    return pow(mix(vec3(grey), texel, SATURATION), vec3(1.0 / LIFT));
}

// The identity up to SHOULDER_FROM, then bends smoothly so that white comes out at BRIGHTEST, keeping its hue.
vec3 shouldered(vec3 colour) {
    float luminance = dot(colour, LUMINANCE);
    if (luminance <= SHOULDER_FROM) return colour;
    float over = luminance - SHOULDER_FROM;
    return colour * ((SHOULDER_FROM + over / (1.0 + BEND * over)) / luminance);
}

void main() {
    vec3 texel = texture2D(map, vUv).rgb;
    vec3 colour = shouldered(softened(texel));

#ifdef SEA_GLINT
    vec3 normal = normalize(vViewNormal);
    float towards = max(dot(normal, normalize(normalize(LIGHT_FROM) + VIEW_DIRECTION)), 0.0);
    float glint = GLINT * pow(towards, GLINT_SHARPNESS) + SHEEN * pow(towards, SHEEN_SHARPNESS);
    colour += GLINT_COLOUR * glint * waterOf(texel);
#endif

    gl_FragColor = vec4(colour, 1.0);
}
