uniform float pointSize;

uniform sampler2D landmassData;
uniform float landmassCount;

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

// Where this tile falls inside the flag painted across its landmass, which flag
// that is, and how much of the landmass its holder actually holds. A zero-width
// region means nobody holds this piece of land.
flat out vec2 vFlagUV;
flat out vec4 vFlagRegion;
flat out float vFlagShare;

void main() {
    vHover = hover;
    vRegionVector = regionVector;

    vFlagRegion = vec4(0.0);
    vFlagUV = vec2(0.0);
    vFlagShare = 0.0;

    if (landmassIndex > 0.5) {
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

            vec3 p = normalize(position);
            float along = dot(p, centre);
            if (along > 0.0) {
                // Distance measured along the surface, not across the chord:
                // the flag is laid on the globe, so it bends with it.
                vec2 offset = vec2(dot(p, east), dot(p, north));
                float reach = length(offset);
                vec2 surface = reach > 1e-6 ? offset / reach * acos(min(along, 1.0)) : vec2(0.0);

                // Left unclamped: the fragment shader clamps it when sampling,
                // so ground past the flag's own rectangle wears the colour the
                // flag ends on instead of falling back to bare Earth.
                vFlagUV = vec2(surface.x / frame.w, surface.y / axis.w) * 0.5 + anchor;
                vFlagRegion = region;
                vFlagShare = share;
            }
        }
    }

    vec3 ground = normalize(position);
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

    gl_PointSize = pointSize * (1.0 + motion * swell);
    gl_Position = projectionMatrix * modelViewMatrix * vec4(displaced, 1.0);
}
