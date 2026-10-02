uniform vec3 colour;
uniform float ink;
uniform float halfWidth;

in float vAcross;

void main() {
    float alpha = clamp(halfWidth - abs(vAcross), 0.0, 1.0);
    if (alpha <= 0.0) discard;
    gl_FragColor = vec4(colour, alpha * ink);
}
