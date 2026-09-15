// One blast's debris, animated entirely here: the CPU sets where and when once,
// and never touches the buffer again.

uniform vec3 centre;
uniform float start;
uniform float radius;
uniform float time;
uniform float pixelsPerRadian;

// x: heading around the centre, y: how far it flies, z: how high it is thrown.
attribute vec3 seed;

varying float vLife;

const float LIFETIME = 1.6;

void main() {
    float s = max(time - start, 0.0);
    float life = clamp(s / LIFETIME, 0.0, 1.0);
    vLife = life;

    vec3 up = centre;
    vec3 east = abs(up.y) > 0.99 ? vec3(1.0, 0.0, 0.0) : normalize(cross(vec3(0.0, 1.0, 0.0), up));
    vec3 north = cross(up, east);
    vec3 heading = east * cos(seed.x) + north * sin(seed.x);

    float out_ = seed.y * radius * 2.4 * (1.0 - exp(-s * 2.8));
    float height = max(0.0, seed.z * radius * 3.0 * s - radius * 2.5 * s * s);

    vec3 p = up * (1.0 + height) + heading * out_;

    gl_PointSize = max(1.5, pixelsPerRadian * radius * 0.09 * (1.0 - life));
    gl_Position = projectionMatrix * modelViewMatrix * vec4(p, 1.0);
}
