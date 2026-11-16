uniform vec3 colour;
uniform float opacity;
uniform float edgeAt;

varying vec2 vUv;

// A soft light along the ring's edge, inside and out, and a faint wash rising toward it from the centre: a halo, never a
// line, so every tile under it stays readable.
void main() {
    vec2 p = (vUv - 0.5) * 2.0 / edgeAt;
    float r = length(p);

    float rim = exp(-pow((r - 1.0) * 6.0, 2.0)) * 0.38;
    float wash = smoothstep(0.3, 1.0, r) * (1.0 - smoothstep(0.98, 1.02, r)) * 0.1;

    float alpha = (rim + wash) * opacity;
    if (alpha < 0.004) discard;

    gl_FragColor = vec4(colour, alpha);
}
