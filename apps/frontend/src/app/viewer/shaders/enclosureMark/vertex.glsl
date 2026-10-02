uniform float tileSize;
uniform float minSize;

attribute float glow;
attribute float scale;
attribute float white;

varying float vGlow;
varying float vWhite;

void main() {
    vGlow = glow;
    vWhite = white;

    gl_PointSize = glow > 0.0 ? max(tileSize, minSize) * scale : 0.0;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
