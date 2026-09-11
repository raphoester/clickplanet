uniform float pixelRatio;

attribute float size;
attribute vec3 tint;

varying vec3 vTint;

void main() {
    vTint = tint;

    gl_PointSize = size * pixelRatio;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
