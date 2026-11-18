uniform vec3 colour;
uniform float ink;
uniform float halfWidth;

in float vAcross;

#ifdef STACKED
flat in float vLayers;
#endif

#if defined(STACKED) || defined(DASHED)
flat in float vLength;
in float vAlong;
#endif

#ifdef DASHED
// 0 solid, 1 dashed: the dashes come in as the gaps fade out. In drawing-buffer pixels, a dash and its gap, and half a
// dash. Must match dashOf in borderLines.test.ts.
uniform float dashed;
uniform float dashPeriod;
uniform float dashHalf;

// A whole number of periods to a piece, a dash centred on each of its ends, so its corners are drawn where a dash turns;
// each period stretched or shrunk to fit, its dash with it. A piece of one period keeps its whole dash: shorter, it is
// mostly dash.
float dashOf(float along, float pieceLength) {
    if (pieceLength < 1.0) return 1.0;
    float count = max(1.0, floor(pieceLength / dashPeriod + 0.5));
    float period = pieceLength / count;
    float halfDash = dashHalf * (count > 1.0 ? period : max(period, dashPeriod)) / dashPeriod;
    float t = along / period;
    return clamp(halfDash + 0.5 - abs(t - floor(t + 0.5)) * period, 0.0, 1.0);
}
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
#ifdef DASHED
    alpha *= mix(1.0, dashOf(vAlong, vLength), dashed);
#endif
    gl_FragColor = vec4(colour, alpha);
}
