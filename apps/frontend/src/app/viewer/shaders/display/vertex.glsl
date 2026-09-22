uniform float pointSize;

uniform sampler2D landmassData;
uniform float landmassCount;

// 1 while the landmass wears its holder's flag, 0 once the tiles speak for
// themselves. Read here only to skip the whole painted-flag lookup below once
// it is zoomed past: the fragment shader mixes it out anyway, and it is four
// vertex texture fetches and a frame per tile to arrive at something nothing
// looks at.
uniform float flagPaint;

// Screen pixels per radian of arc at the current zoom.
uniform float pixelsPerRadian;

// Bombs. Matches MAX_BLASTS and BLAST_TIMELINE in domain/blast.ts.
#define MAX_BLASTS 4
const float BLAST_FALL = 0.8;
const float BLAST_SCORCH = 5.0;

uniform float time;
// xyz: where it lands, on the unit sphere. w: when it was dropped, in seconds.
uniform vec4 blasts[MAX_BLASTS];
// Radians of arc; zero for a slot with nothing in it.
uniform float blastRadii[MAX_BLASTS];
// Zero under prefers-reduced-motion: the colours stay, nothing moves.
uniform float motion;

// The target rings are not drawn here but as meshes (blasts.ts): the tiles are
// dots with sea between them, and a ring made of dots breaks up as it moves.
varying float vGlow;
varying float vScorch;

attribute float hover;
attribute vec4 regionVector;
attribute float landmassIndex;

varying float vHover;
flat out vec4 vRegionVector;

// The flag painted across this tile's landmass: which flag it is, how much of
// the landmass its holder actually holds, and — rather than one colour for the
// whole tile — the frame the fragment shader reads it in. A zero-width region
// means nobody holds this piece of land.
//
// `vFlagUV` is where the tile's own centre falls in that flag and `vFlagStep`
// is how far that slides under one screen pixel, across and up. Together they
// are the flag as a function of the ground under the pixel, which is what lets
// the discs — which overlap, because they have to cover the ground for the
// flag to reach it — agree with each other. Handing each disc a single sample
// instead made the whole painted flag a mosaic at the resolution of the tile
// lattice: fine for bands, and pixel soup for any flag carrying a device, on
// exactly the landmasses that are only a few tiles across.
flat out vec2 vFlagUV;
flat out vec4 vFlagStep;
flat out vec4 vFlagRegion;
flat out float vFlagShare;

// The size the disc is actually drawn at, which a blast swells. The fragment
// shader needs it to turn `gl_PointCoord` back into screen pixels.
flat out float vSpriteSize;

/**
 * Where a point of ground falls inside the flag painted across its landmass,
 * with distance measured *along the surface* so the flag bends with the globe.
 *
 * Left unclamped: the fragment shader clamps it when sampling, so ground past
 * the flag's own rectangle wears the colour the flag ends on instead of
 * falling back to bare Earth.
 */
vec2 flagUVof(vec3 ground, vec3 centre, vec3 east, vec3 north, vec2 halfSize, vec2 anchor) {
    vec2 offset = vec2(dot(ground, east), dot(ground, north));
    float reach = length(offset);
    float angle = acos(clamp(dot(ground, centre), -1.0, 1.0));
    vec2 surface = reach > 1e-6 ? offset / reach * angle : vec2(0.0);
    return surface / halfSize * 0.5 + anchor;
}

/**
 * The step across the ground that moves a point one screen pixel along `screenAxis`,
 * one of the camera's own axes in the object space the tiles live in.
 *
 * It is not simply the pixel's worth of `screenAxis`: only the part of a ground
 * step that survives the projection counts, so the tangent part is divided by
 * how much of itself the projection keeps. That divisor runs to zero at the
 * limb, where the globe is edge-on and one pixel covers the rest of it, so it
 * is floored — inside that floor the flag is a foreshortened sliver either way,
 * and without it the step would be infinite.
 */
vec3 screenStep(vec3 ground, vec3 screenAxis, float perPixel) {
    float along = dot(ground, screenAxis);
    return (screenAxis - along * ground) * (perPixel / max(1.0 - along * along, 0.05));
}

/**
 * How far past the globe's limb a tile can still be seen, as the sine of that
 * angle. The Earth is opaque at 0.999 and the tiles sit at 1, so a tile up to
 * `acos(0.999 / 1.0)` — about 0.045 — past the limb still shows against the
 * sky rather than being covered by the Earth's own silhouette.
 */
const float LIMB = 0.05;

void main() {
    vec3 ground = normalize(position);

    // The far side of the globe is covered by the opaque Earth, so every tile
    // there was run through the whole landmass lookup below and then thrown
    // away by the depth test — half the field, on every frame, four vertex
    // texture fetches each. One dot product takes them out before any of it.
    //
    // What may not be taken out is everything the Earth's silhouette does not
    // cover: `LIMB` past the limb, plus half of the tile's own disc, plus how
    // far a blast may throw it outward (`motion` is zero under reduced motion,
    // where nothing is displaced at all).
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

    vFlagRegion = vec4(0.0);
    vFlagUV = vec2(0.0);
    vFlagStep = vec4(0.0);
    vFlagShare = 0.0;

    if (flagPaint > 0.0 && landmassIndex > 0.5) {
        float row = (landmassIndex + 0.5) / landmassCount;
        vec4 region = texture(landmassData, vec2(5.0 / 8.0, row));
        vec4 held = texture(landmassData, vec2(7.0 / 8.0, row));

        // A painted flag has to earn its place on screen. Andorra is one tile:
        // from orbit it would be a single hyper-bright speck, so a flag fades in
        // only once its landmass is big enough to read, and the same rule
        // quietly clears the oceans of lone islands.
        float share = held.r * smoothstep(5.0, 16.0, held.g * pixelsPerRadian);
        vec2 anchor = held.ba;

        if (region.z > 0.0 && share > 0.0) {
            vec4 frame = texture(landmassData, vec2(1.0 / 8.0, row));
            vec4 axis = texture(landmassData, vec2(3.0 / 8.0, row));
            vec3 centre = frame.xyz;
            vec3 east = normalize(vec3(centre.z, 0.0, -centre.x));
            if (abs(centre.y) > 0.9999) east = vec3(1.0, 0.0, 0.0);
            vec3 north = cross(centre, east);
            vec2 halfSize = vec2(frame.w, axis.w);

            // The far half of the globe has no business in this frame: past a
            // quarter turn the surface distance stops naming a point on the
            // flag at all, and clamping it would paint the flag's edge colour
            // across the other side of the planet.
            if (dot(ground, centre) > 0.0) {
                // The camera's own axes, in the object space the tiles live in:
                // the rows of the model-view rotation, which is orthonormal.
                vec3 right = normalize(vec3(modelViewMatrix[0][0], modelViewMatrix[1][0], modelViewMatrix[2][0]));
                vec3 up = normalize(vec3(modelViewMatrix[0][1], modelViewMatrix[1][1], modelViewMatrix[2][1]));

                // The camera is orthographic against a globe of radius 1, so a
                // world unit at the surface is a radian of arc and `pixelsPerRadian`
                // is also pixels per world unit.
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

        // Nothing past three radii ever lights up, and that is nearly every
        // tile on the planet: one dot product and out.
        float along = dot(ground, blasts[i].xyz);
        if (along < cos(min(radius * 3.0, 3.14159))) continue;

        // In radii from the centre: 1.0 is the edge of what was cleared.
        float d = acos(min(along, 1.0)) / radius;

        // Still falling: the ground is untouched until it lands.
        if (t < BLAST_FALL) continue;

        float s = t - BLAST_FALL;

        // The shock front runs out past the crater and dies on the way.
        float front = 2.6 * (1.0 - exp(-s * 3.2));
        float ring = exp(-pow((d - front) * 2.8, 2.0)) * exp(-s * 1.6);
        float flash = (1.0 - smoothstep(0.0, 1.1, d)) * exp(-s * 5.0);
        vGlow = max(vGlow, max(ring, flash));

        // Tiles are thrown out along the ground by the front, and the crater
        // swells toward the viewer as it flashes.
        vec3 outward = ground - blasts[i].xyz * along;
        float reach = length(outward);
        if (reach > 1e-5) displaced += motion * (outward / reach) * ring * radius * 0.3;
        displaced += motion * ground * flash * radius * 0.25;
        swell = max(swell, ring * 1.2 + flash * 1.8);

        // The crater itself stays burnt, cooling over the scorch.
        float crater = 1.0 - smoothstep(0.85, 1.1, d);
        vScorch = max(vScorch, crater * (1.0 - smoothstep(0.0, BLAST_SCORCH, s)));
    }

    vSpriteSize = pointSize * (1.0 + motion * swell);
    gl_PointSize = vSpriteSize;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(displaced, 1.0);
}
