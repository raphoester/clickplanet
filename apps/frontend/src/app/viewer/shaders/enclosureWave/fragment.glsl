uniform vec3 colour;
uniform float radius;
uniform float opacity;
uniform float rim;

varying vec2 vPlane;

const float THICKNESS = 0.14;

void main() {
    float d = length(vPlane);
    if (d > 1.0) discard;

    float ring = smoothstep(radius - THICKNESS, radius, d) * (1.0 - smoothstep(radius, radius + THICKNESS * 0.5, d));
    float wash = 0.18 * (1.0 - smoothstep(0.0, radius, d));
    float light = ring + wash;

    // A dark edge on both sides of the ring, so it reads on a white flag as well as on the sea.
    float edge = rim * smoothstep(radius - THICKNESS * 1.8, radius - THICKNESS, d) * (1.0 - smoothstep(radius + THICKNESS * 0.5, radius + THICKNESS * 1.1, d));
    float alpha = max(light, edge);

    gl_FragColor = vec4(colour * (light / max(alpha, 1e-4)), clamp(alpha * opacity, 0.0, 1.0));
}
