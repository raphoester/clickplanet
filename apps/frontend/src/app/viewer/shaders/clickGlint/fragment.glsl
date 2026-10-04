uniform vec3 colour;
uniform float opacity;
uniform float white;

void main() {
    float d = length(gl_PointCoord - vec2(0.5)) * 2.0;
    if (d > 1.0) discard;

    float glow = exp(-d * d * 3.0) * (1.0 - smoothstep(0.8, 1.0, d));

    gl_FragColor = vec4(mix(colour, vec3(1.0), white), clamp(glow * opacity, 0.0, 1.0));
}
