uniform vec3 colour;
uniform float ink;
uniform float halfWidth;

in float vAcross;

#ifdef STACKED
flat in float vLayers;
flat in float vLength;
in float vAlong;
#endif

void main() {
    float alpha = clamp(halfWidth - abs(vAcross), 0.0, 1.0);
    if (alpha <= 0.0) discard;
    alpha *= ink;
#ifdef STACKED
    // A coarse piece inks as its cell edges' quads laid over one another; where two pieces' ends overlap, each half.
    float covering = 1.0 + step(vAlong, halfWidth) + step(vLength - halfWidth, vAlong);
    alpha = 1.0 - pow(1.0 - alpha, vLayers / covering);
#endif
    gl_FragColor = vec4(colour, alpha);
}
