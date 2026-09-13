// A ring laid flat on the ground, running out from the middle of a closed shape.
uniform vec3 colour;
uniform float radius;
uniform float opacity;

varying vec2 vPlane;

const float THICKNESS = 0.14;

void main() {
    float d = length(vPlane);
    if (d > 1.0) discard;

    float ring = smoothstep(radius - THICKNESS, radius, d) * (1.0 - smoothstep(radius, radius + THICKNESS * 0.5, d));
    // A faint wash inside the ring, so the ground it passed over reads as swept.
    float wash = 0.18 * (1.0 - smoothstep(0.0, radius, d));

    gl_FragColor = vec4(colour, clamp((ring + wash) * opacity, 0.0, 1.0));
}
