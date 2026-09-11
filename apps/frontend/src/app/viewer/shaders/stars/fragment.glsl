varying vec3 vTint;

void main() {
    float distance = length(gl_PointCoord - vec2(0.5));

    // A star is one or two pixels across, so the fade has to be kept to the
    // last of them: spread any wider and the whole star is drawn at the
    // falloff's strength rather than at its own.
    float alpha = 1.0 - smoothstep(0.3, 0.5, distance);
    if (alpha <= 0.0) discard;

    gl_FragColor = vec4(vTint, alpha);
}
