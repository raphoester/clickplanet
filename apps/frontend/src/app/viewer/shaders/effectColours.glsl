vec4 scorched(vec4 colour, float scorch) {
    vec3 ember = mix(vec3(0.07, 0.02, 0.01), vec3(1.0, 0.32, 0.04), scorch * scorch);
    return mix(colour, vec4(ember, 0.95), scorch * 0.9);
}

vec4 steeled(vec4 colour, float fortify) {
    vec3 steel = mix(vec3(0.55, 0.72, 0.95), vec3(0.96, 0.99, 1.0), fortify * fortify);
    return mix(colour, vec4(steel, 1.0), fortify * 0.85);
}

vec4 heated(vec4 colour, float glow) {
    vec3 hot = mix(vec3(1.0, 0.42, 0.08), vec3(1.0, 0.96, 0.82), glow * glow);
    return mix(colour, vec4(hot, 1.0), glow);
}
