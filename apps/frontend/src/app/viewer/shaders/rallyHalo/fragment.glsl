uniform vec3 colour;
uniform float opacity;
uniform float edgeAt;
// The share of its time the rally has left: 1 as it is called, 0 at its end. The rim glows along that much of the ring
// from its north, clockwise, and the time gone keeps a faint trace, so the ring is the rally's clock.
uniform float left;

varying vec2 vUv;

const float TAU = 6.2831853;

// A soft light along the ring's edge, inside and out, and a faint wash rising toward it from the centre: a halo, never a
// line, so every tile under it stays readable.
void main() {
    vec2 p = (vUv - 0.5) * 2.0 / edgeAt;
    float r = length(p);
    float turn = fract(atan(p.x, p.y) / TAU);

    float band = exp(-pow((r - 1.0) * 6.0, 2.0));
    float lit = 1.0 - smoothstep(left - 0.003, left + 0.003, turn);
    float head = exp(-pow((turn - left) * 45.0, 2.0)) * step(0.001, left);
    float rim = band * (mix(0.08, 0.42, lit) + 0.35 * head);
    float wash = smoothstep(0.3, 1.0, r) * (1.0 - smoothstep(0.98, 1.02, r)) * 0.1;

    float alpha = (rim + wash) * opacity;
    if (alpha < 0.004) discard;

    gl_FragColor = vec4(colour, alpha);
}
