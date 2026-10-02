varying vec3 vTint;

void main() {
    float distance = length(gl_PointCoord - vec2(0.5));

    float alpha = 1.0 - smoothstep(0.3, 0.5, distance);
    if (alpha <= 0.0) discard;

    gl_FragColor = vec4(vTint, alpha);
}
