uniform float pointSize;

attribute vec3 color;
varying vec3 vColor;

void main() {
    vColor = color;

    gl_PointSize = pointSize;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
