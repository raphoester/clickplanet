const vec3 LIGHT_FROM = vec3(-0.45, 0.55, 0.70);

const float SHADOW = 0.42;

// Screen colour, not linear: a raw ShaderMaterial gets no colour management.
const vec3 AIR = vec3(0.30, 0.62, 1.0);
const float AIR_AT_LIMB = 0.5;

float lightOf(vec3 normal) {
    float facing = dot(normal, normalize(LIGHT_FROM)) * 0.5 + 0.5;
    return mix(SHADOW, 1.0, facing * facing);
}

float shadeOf(vec3 normal) {
    return lightOf(normal) / lightOf(vec3(0.0, 0.0, 1.0));
}

float hazeOf(vec3 normal) {
    float edgeOn = 1.0 - clamp(normal.z, 0.0, 1.0);
    return AIR_AT_LIMB * edgeOn * edgeOn * edgeOn;
}

vec3 lit(vec3 colour, float shade, float haze) {
    return mix(colour, AIR, haze) * shade;
}
