uniform float pointSize;

uniform sampler2D landmassData;
uniform float landmassCount;

// Screen pixels per radian of arc at the current zoom.
uniform float pixelsPerRadian;

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

    gl_PointSize = pointSize;
    gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
}
