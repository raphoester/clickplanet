uniform vec3 colour;

varying float vGlow;
varying float vWhite;

void main() {
    float d = length(gl_PointCoord - vec2(0.5)) * 2.0;
    if (d > 1.0) discard;

    // A solid disc with a soft edge, drawn over the flag rather than added to
    // it: added light vanishes into a white stripe, and half the flags on the
    // map have one.
    float disc = 1.0 - smoothstep(0.7, 1.0, d);

    gl_FragColor = vec4(mix(colour, vec3(1.0), clamp(vWhite, 0.0, 1.0)), clamp(disc * vGlow, 0.0, 1.0));
}
