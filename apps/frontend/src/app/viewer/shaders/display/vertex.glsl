#ifdef LIT
#include ../light.glsl;
#endif

uniform float pointSize;

uniform sampler2D landmassData;
uniform float landmassCount;

uniform float flagPaint;

uniform float pixelsPerRadian;

// Must match MAX_BLASTS and BLAST_TIMELINE in domain/blast.ts.
#define MAX_BLASTS 4
const float BLAST_FALL = 0.8;
const float BLAST_SCORCH = 5.0;

uniform float time;
uniform vec4 blasts[MAX_BLASTS];
uniform float blastRadii[MAX_BLASTS];
uniform float motion;

varying float vGlow;
varying float vScorch;

attribute float hover;
attribute vec4 regionVector;
attribute float landmassIndex;
attribute float garrison;

varying float vHover;
flat out vec4 vRegionVector;
flat out float vGarrison;

#ifdef LIT
flat out float vShade;
flat out float vHaze;
#endif

flat out vec2 vFlagUV;
flat out vec4 vFlagStep;
flat out vec4 vFlagRegion;
flat out float vFlagShare;

flat out float vSpriteSize;

vec2 flagUVof(vec3 ground, vec3 centre, vec3 east, vec3 north, vec2 halfSize, vec2 anchor) {
    vec2 offset = vec2(dot(ground, east), dot(ground, north));
    float reach = length(offset);
    float angle = acos(clamp(dot(ground, centre), -1.0, 1.0));
    vec2 surface = reach > 1e-6 ? offset / reach * angle : vec2(0.0);
    return surface / halfSize * 0.5 + anchor;
}

vec3 screenStep(vec3 ground, vec3 screenAxis, float perPixel) {
    float along = dot(ground, screenAxis);
    return (screenAxis - along * ground) * (perPixel / max(1.0 - along * along, 0.05));
}

const float LIMB = 0.05;

void main() {
    vec3 ground = normalize(position);

    float thrown = 0.0;
    for (int i = 0; i < MAX_BLASTS; i++) thrown = max(thrown, blastRadii[i]);
    float limb = LIMB + (pointSize * 0.5 + 1.0) / max(pixelsPerRadian, 1.0) + motion * thrown * 0.6;

    if ((normalMatrix * ground).z < -limb) {
        gl_Position = vec4(2.0, 2.0, 2.0, 1.0);
        gl_PointSize = 0.0;
        vSpriteSize = 0.0;
        return;
    }

    vHover = hover;
    vRegionVector = regionVector;
    vGarrison = garrison;

#ifdef LIT
    vec3 normal = normalize(normalMatrix * ground);
    vShade = shadeOf(normal);
    vHaze = hazeOf(normal);
#endif

    vFlagRegion = vec4(0.0);
    vFlagUV = vec2(0.0);
    vFlagStep = vec4(0.0);
    vFlagShare = 0.0;

    if (flagPaint > 0.0 && landmassIndex > 0.5) {
        float row = (landmassIndex + 0.5) / landmassCount;
        vec4 region = texture(landmassData, vec2(5.0 / 8.0, row));
        vec4 held = texture(landmassData, vec2(7.0 / 8.0, row));

        float share = held.r;
        vec2 anchor = held.ba;

        if (region.z > 0.0 && share > 0.0) {
            vec4 frame = texture(landmassData, vec2(1.0 / 8.0, row));
            vec4 axis = texture(landmassData, vec2(3.0 / 8.0, row));
            vec3 centre = frame.xyz;
            vec3 east = normalize(vec3(centre.z, 0.0, -centre.x));
            if (abs(centre.y) > 0.9999) east = vec3(1.0, 0.0, 0.0);
            vec3 north = cross(centre, east);
            vec2 halfSize = vec2(frame.w, axis.w);

            if (dot(ground, centre) > 0.0) {
                vec3 right = normalize(vec3(modelViewMatrix[0][0], modelViewMatrix[1][0], modelViewMatrix[2][0]));
                vec3 up = normalize(vec3(modelViewMatrix[0][1], modelViewMatrix[1][1], modelViewMatrix[2][1]));

                float perPixel = 1.0 / max(pixelsPerRadian, 1.0);

                vec2 uv = flagUVof(ground, centre, east, north, halfSize, anchor);
                vec3 across = normalize(ground + screenStep(ground, right, perPixel));
                vec3 upward = normalize(ground + screenStep(ground, up, perPixel));

                vFlagUV = uv;
                vFlagStep = vec4(
                    flagUVof(across, centre, east, north, halfSize, anchor) - uv,
                    flagUVof(upward, centre, east, north, halfSize, anchor) - uv
                );
                vFlagRegion = region;
                vFlagShare = share;
            }
        }
    }

    vec3 displaced = position;
    float swell = 0.0;

    vGlow = 0.0;
    vScorch = 0.0;

    for (int i = 0; i < MAX_BLASTS; i++) {
        float radius = blastRadii[i];
        if (radius <= 0.0) continue;

        float t = time - blasts[i].w;
        if (t < 0.0 || t > BLAST_FALL + BLAST_SCORCH) continue;

        float along = dot(ground, blasts[i].xyz);
        if (along < cos(min(radius * 3.0, 3.14159))) continue;

        float d = acos(min(along, 1.0)) / radius;

        if (t < BLAST_FALL) continue;

        float s = t - BLAST_FALL;

        float front = 2.6 * (1.0 - exp(-s * 3.2));
        float ring = exp(-pow((d - front) * 2.8, 2.0)) * exp(-s * 1.6);
        float flash = (1.0 - smoothstep(0.0, 1.1, d)) * exp(-s * 5.0);
        vGlow = max(vGlow, max(ring, flash));

        vec3 outward = ground - blasts[i].xyz * along;
        float reach = length(outward);
        if (reach > 1e-5) displaced += motion * (outward / reach) * ring * radius * 0.3;
        displaced += motion * ground * flash * radius * 0.25;
        swell = max(swell, ring * 1.2 + flash * 1.8);

        float crater = 1.0 - smoothstep(0.85, 1.1, d);
        vScorch = max(vScorch, crater * (1.0 - smoothstep(0.0, BLAST_SCORCH, s)));
    }

    vSpriteSize = pointSize * (1.0 + motion * swell);
    gl_PointSize = vSpriteSize;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(displaced, 1.0);
}
