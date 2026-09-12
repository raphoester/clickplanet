// A target ring laid flat on the globe. Drawn per pixel rather than out of the
// tile discs, so it stays whole over the sea and between tiles, and its edge is
// smooth wherever it moves.

uniform vec3 colour;
uniform float opacity;
// How much of the quad's half-width the ring's edge sits at.
uniform float edgeAt;
// A held press on its way to dropping the bomb: the edge fills in clockwise and
// the inside brightens, so letting go early reads as backing out of something.
uniform float progress;

varying vec2 vUv;

const float TAU = 6.2831853;

void main() {
    vec2 p = (vUv - 0.5) * 2.0 / edgeAt;
    float r = length(p);

    float edge = exp(-pow((r - 1.0) * 8.0, 2.0));
    float fill = (1.0 - smoothstep(0.8, 1.0, r)) * (0.16 + 0.3 * progress);

    float turn = fract(atan(p.x, p.y) / TAU);
    float charged = progress > 0.0 ? step(turn, progress) * exp(-pow((r - 1.0) * 4.5, 2.0)) : 0.0;

    vec3 hot = mix(colour, vec3(1.0, 0.92, 0.6), charged);
    float alpha = max(max(edge, fill), charged) * opacity;
    if (alpha < 0.004) discard;

    gl_FragColor = vec4(hot, alpha);
}
