uniform vec3 colour;
uniform float ink;
uniform float halfWidth;

in float vAcross;

void main() {
    // One pixel of feather on each side, so the line has the same weight
    // wherever it runs rather than crawling as the globe turns.
    float alpha = clamp(halfWidth - abs(vAcross), 0.0, 1.0);
    if (alpha <= 0.0) discard;
    gl_FragColor = vec4(colour, alpha * ink);
}
