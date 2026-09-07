uniform float pointSize;

attribute float hover;
attribute vec4 regionVector;

varying float vHover;
flat out vec4 vRegionVector;

void main() {
    vHover = hover;
    vRegionVector = regionVector;

    gl_PointSize = pointSize;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
