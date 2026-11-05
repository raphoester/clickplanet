uniform vec3 skin;
uniform vec3 flesh;
uniform vec3 candle;
uniform float flicker;
uniform float carved;
uniform float opacity;
uniform sampler2D carving;

varying vec3 vViewNormal;
varying vec2 vUv;

const vec3 LIGHT = vec3(-0.45, 0.65, 0.61);

void main() {
    vec3 n = normalize(vViewNormal);
    vec3 light = normalize(LIGHT);
    float lambert = max(dot(n, light), 0.0);
    float rim = pow(1.0 - max(n.z, 0.0), 2.0);
    float sheen = pow(max(reflect(-light, n).z, 0.0), 18.0) * 0.3;
    vec3 colour = skin * (0.4 + 0.75 * lambert) * (1.0 - 0.35 * rim) + sheen;

    // The mask is blurred: its middle value is the cut, below it the shadow round the cut, above it the flesh then the light.
    float cut = carved * texture2D(carving, vUv).r;
    colour *= 1.0 - 0.6 * smoothstep(0.05, 0.5, cut);
    colour = mix(colour, flesh, smoothstep(0.46, 0.54, cut));
    colour = mix(colour, candle * flicker, smoothstep(0.6, 0.85, cut));

    gl_FragColor = vec4(colour, opacity);
}
