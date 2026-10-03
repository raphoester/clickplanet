uniform float size;
uniform float unitsPerPixel;

void main() {
    gl_PointSize = size;

    // A sprite has one depth: without this pull toward the camera, the curve of the globe hides part of it.
    vec4 view = modelViewMatrix * vec4(position, 1.0);
    view.z += size * unitsPerPixel;
    gl_Position = projectionMatrix * view;
}
