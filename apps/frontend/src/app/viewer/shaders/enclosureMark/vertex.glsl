uniform float tileSize;
uniform float minSize;
uniform float unitsPerPixel;

attribute float glow;
attribute float scale;
attribute float white;

varying float vGlow;
varying float vWhite;

void main() {
    vGlow = glow;
    vWhite = white;

    float size = glow > 0.0 ? max(tileSize, minSize) * scale : 0.0;
    gl_PointSize = size;

    // A sprite has one depth: without this pull toward the camera, the curve of the globe hides part of it.
    vec4 view = modelViewMatrix * vec4(position, 1.0);
    view.z += size * unitsPerPixel;
    gl_Position = projectionMatrix * view;
}
