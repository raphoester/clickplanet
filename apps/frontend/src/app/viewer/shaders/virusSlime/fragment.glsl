// A virus's goo: blobs melted together (a source, a neck, a head, droplets), lit as a glossy dome with bubbles turning
// inside, a dark skin at its edge and a slow mottle; a cured tile's dries to a flat grey husk. Lengths are in tile
// spacings; grow scales the goo, never the way between its tiles, so from orbit it is still seen.
uniform float time;
uniform float grow;
// A drawing-buffer pixel, in tile spacings.
uniform float unit;

varying vec2 vLocal;
flat in float vLength;
// along, head, source, neck
flat in vec4 vShape;
// drops, drop, wobble, spray
flat in vec4 vSplash;
// alpha, dead, gloss, flash
flat in vec4 vTone;
flat in float vSeed;
flat in vec3 vLight;
flat in vec3 vView;

// Must match the infected tile's in the display fragment shader, and --virus.
const vec3 VIRUS = vec3(0.608, 0.302, 0.878);
const vec3 VIRUS_LIGHT = vec3(0.827, 0.651, 1.0);
const vec3 VIRUS_DARK = vec3(0.23, 0.07, 0.34);
// What a cured tile's goo dries to.
const vec3 HUSK = vec3(0.4, 0.37, 0.36);

const int DROPS = 7;
const int BUBBLES = 3;
// How thick the goo is, from its edge in: the dome's slope.
const float THICKNESS = 0.22;
const float SKIN = 0.06;

float hash(float n) {
    return fract(sin(n) * 43758.5453);
}

float hash2(vec2 p) {
    return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
}

float noise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    vec2 u = f * f * (3.0 - 2.0 * f);
    return mix(mix(hash2(i), hash2(i + vec2(1.0, 0.0)), u.x), mix(hash2(i + vec2(0.0, 1.0)), hash2(i + vec2(1.0, 1.0)), u.x), u.y);
}

// The smooth minimum: two blobs closer than k melt into one.
float smin(float a, float b, float k) {
    float h = clamp(0.5 + 0.5 * (b - a) / k, 0.0, 1.0);
    return mix(b, a, h) - k * h * (1.0 - h);
}

vec2 sourceAt() {
    return vec2(-0.5 * vLength, 0.0);
}

vec2 headAt() {
    return vec2((vShape.x - 0.5) * vLength, 0.0);
}

// The distance to the goo's edge, negative inside.
float goo(vec2 p) {
    vec2 source = sourceAt();
    vec2 head = headAt();
    float k = 0.16 * grow;
    float d = 1e3;

    if (vShape.z > 0.0) d = length(p - source) - vShape.z * grow;

    if (vShape.w > 0.0 && vShape.x > 0.0) {
        vec2 ba = head - source;
        vec2 pa = p - source;
        float h = clamp(dot(pa, ba) / max(dot(ba, ba), 1e-6), 0.0, 1.0);
        // Thicker where it pulls out of its source.
        d = smin(d, length(pa - ba * h) - vShape.w * grow * mix(1.25, 0.8, h), k * 0.6);
    }

    vec2 q = p - head;
    if (vShape.y > 0.0) {
        float angle = atan(q.y, q.x);
        // Lobes of no common count, each turned by the seed, so no two blobs wobble into the same star.
        float jiggle = 0.35 * sin(angle * 2.0 + vSeed)
            + 0.5 * sin(angle * 3.0 - time * 6.5 + vSeed * 2.3)
            + 0.3 * sin(angle * 5.0 + time * 9.0 + vSeed * 3.7)
            + 0.15 * sin(angle * 7.0 - time * 11.0 + vSeed * 5.1);
        d = smin(d, length(q) - vShape.y * grow * (1.0 + 0.06 * vSplash.z * jiggle), k);
    }

    if (vSplash.y > 0.0) {
        for (int i = 0; i < DROPS; i++) {
            float fi = float(i);
            float around = (fi + 0.5) / float(DROPS) - 0.5;
            // Ahead of the head, in a cone, or all round it.
            float angle = around * mix(2.8, 6.2831853, vSplash.w) + (hash(vSeed + fi * 1.3) - 0.5) * 0.6;
            float far = vSplash.x * grow * (0.7 + 0.6 * hash(vSeed * 3.1 + fi));
            float size = vSplash.y * grow * (0.6 + 0.8 * hash(vSeed * 7.3 + fi));
            d = smin(d, length(q - vec2(cos(angle), sin(angle)) * far) - size, 0.07 * grow);
        }
    }
    return d;
}

float dome(float d) {
    return sqrt(clamp(-d / (THICKNESS * grow), 0.0, 1.0));
}

void main() {
    vec2 p = vLocal;
    float d = goo(p);
    float aa = unit;
    if (d > aa) discard;
    float cover = 1.0 - smoothstep(-aa, aa, d);

    // The dome's normal, from its height a step across and a step along.
    float delta = max(unit, 0.015 * grow);
    float h = dome(d);
    float slope = THICKNESS * grow * vTone.z;
    vec3 normal = normalize(vec3(
        -slope * (dome(goo(p + vec2(delta, 0.0))) - h) / delta,
        -slope * (dome(goo(p + vec2(0.0, delta))) - h) / delta,
        1.0));
    vec3 light = normalize(vLight);
    vec3 viewer = normalize(vView);
    vec3 halfway = normalize(light + viewer);

    float lit = 0.62 + 0.38 * max(dot(normal, light), 0.0);
    float spec = pow(max(dot(normal, halfway), 0.0), 60.0);
    float sheen = pow(max(dot(normal, halfway), 0.0), 10.0);
    float rim = pow(1.0 - max(dot(normal, viewer), 0.0), 3.0);

    // Thin goo is light and clear, thick goo deep; a slow mottle of its own in it.
    vec3 colour = mix(VIRUS_LIGHT, VIRUS, smoothstep(0.0, 0.7, h));
    float mottle = noise(p / grow * 2.6 + vec2(vSeed, time * 0.25)) * 0.65 + noise(p / grow * 6.0 - vec2(time * 0.4, vSeed)) * 0.35;
    colour = mix(colour, VIRUS_DARK, 0.4 * mottle * h);
    colour *= lit;

    // Bubbles turning inside the head, once it is big enough on screen to hold them.
    float radius = vShape.y * grow;
    float bubbles = smoothstep(7.0, 14.0, radius / unit) * vTone.z * (1.0 - vTone.y);
    if (bubbles > 0.0 && h > 0.2) {
        vec2 q = p - headAt();
        vec2 towardLight = normalize(vLight.xy + vec2(1e-4));
        for (int i = 0; i < BUBBLES; i++) {
            float fi = float(i);
            float angle = vSeed * 1.7 + fi * 2.1 + time * (0.4 + 0.5 * hash(vSeed + fi));
            vec2 centre = vec2(cos(angle), sin(angle)) * radius * (0.18 + 0.35 * hash(vSeed * 1.9 + fi * 5.0));
            float size = radius * (0.08 + 0.07 * hash(vSeed * 2.3 + fi));
            float apart = length(q - centre);
            float inside = 1.0 - smoothstep(size - aa, size, apart);
            float ring = smoothstep(size - 1.5 * aa, size, apart) * (1.0 - smoothstep(size, size + aa, apart));
            float glint = 1.0 - smoothstep(0.0, size * 0.35, length(q - centre - towardLight * size * 0.45));
            colour = mix(colour, VIRUS_LIGHT, inside * 0.35 * bubbles);
            colour = mix(colour, vec3(1.0), (ring * 0.5 + glint * 0.8) * bubbles);
        }
    }

    // A dark skin at the edge, so the goo reads on every flag.
    float skin = smoothstep(-SKIN * grow - aa, 0.0, d);
    colour = mix(colour, VIRUS_DARK, skin * 0.8);

    float gloss = vTone.z * (1.0 - vTone.y);
    colour = mix(colour, HUSK * (0.75 + 0.35 * mottle), vTone.y);
    colour += vec3(0.9 * spec + 0.1 * sheen) * gloss + VIRUS_LIGHT * 0.25 * rim * gloss;
    colour = mix(colour, vec3(1.0), 0.55 * vTone.w);

    float body = mix(0.62, 0.95, smoothstep(0.0, 0.6, h));
    gl_FragColor = vec4(colour, cover * vTone.x * max(body, skin * 0.95));
}
