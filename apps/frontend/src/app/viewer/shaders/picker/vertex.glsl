uniform float pointSize;

#ifndef COMPACT
attribute vec3 color;
#endif

varying vec3 vColor;

void main() {
#ifdef COMPACT
    // Must match integerToColor in pickingColors.ts: tile id = vertex index + 1.
    int id = gl_VertexID + 1;
    vColor = vec3((id >> 16) & 255, (id >> 8) & 255, id & 255) / 255.0;
#else
    vColor = color;
#endif

    gl_PointSize = pointSize;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
