// A tile of a closed shape, lit up. Everything that moves is written per frame
// from enclosureEffect.ts; this only turns it into a disc on screen.
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

    // Never smaller than minSize, so a shape closed on the far side of the zoom
    // is still a spark rather than a pixel lost among a quarter of a million.
    gl_PointSize = glow > 0.0 ? max(tileSize, minSize) * scale : 0.0;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
