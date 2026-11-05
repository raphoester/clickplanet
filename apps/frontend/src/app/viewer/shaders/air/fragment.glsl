uniform vec3 colour;
uniform vec3 sky;
uniform vec2 centre;
uniform float radius;

#include ../light.glsl;

// 1 at the middle of the disc, so a player at work zoomed in sees the flags as they are.
const float SHADOW = 0.6;
const float LIGHT = 0.55;
const float RIM = 0.35;
const float HALO = 0.55;
const float HALO_FALLOFF = 18.0;

// Blended as src + dst * src.a: alpha shades what is drawn, the colour is the air laid over it.
void main() {
    vec2 at = (gl_FragCoord.xy - centre) / radius;
    float reach = dot(at, at);
    if (reach < 1.0) {
        float facing = sqrt(1.0 - reach);
        float shade = min(1.0, SHADOW + LIGHT * max(dot(vec3(at, facing), normalize(LIGHT_FROM)), 0.0));
        float rim = pow(1.0 - facing, 3.0);
        gl_FragColor = vec4(colour * RIM * rim, shade * (1.0 - RIM * rim));
    } else {
        float halo = exp(-(sqrt(reach) - 1.0) * HALO_FALLOFF);
        gl_FragColor = vec4(max(colour - sky, 0.0) * HALO * halo, 1.0);
    }
}
