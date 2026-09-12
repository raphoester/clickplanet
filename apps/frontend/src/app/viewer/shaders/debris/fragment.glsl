// 1 for a bomb that fell in the sea: spray rather than sparks.
uniform float water;

varying float vLife;

void main() {
    float d = length(gl_PointCoord - vec2(0.5));
    if (d > 0.5) discard;

    // White-hot sparks cooling through orange to dull red.
    vec3 colour = mix(vec3(1.0, 0.93, 0.7), vec3(1.0, 0.35, 0.05), smoothstep(0.0, 0.35, vLife));
    colour = mix(colour, vec3(0.45, 0.06, 0.02), smoothstep(0.35, 1.0, vLife));

    vec3 spray = mix(vec3(0.95, 0.99, 1.0), vec3(0.35, 0.65, 0.95), smoothstep(0.0, 1.0, vLife));
    colour = mix(colour, spray, water);

    float alpha = (1.0 - vLife) * (1.0 - smoothstep(0.2, 0.5, d));
    gl_FragColor = vec4(colour * alpha, alpha);
}
