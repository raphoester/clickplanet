// The one light the globe is lit by, shared by everything drawn on its surface
// — the earth and the tiles over it — so a flag and the ground beside it are
// always in the same light. Everything here is in view space, where the camera
// looks down -Z.
//
// It is a studio light and not the sun: it sits with the camera, up and to the
// left, so the same side of the globe is always lit whichever way it is turned.
// A real sun would put half the players' countries in the dark at any moment.

// Where the light comes from: up, left, and well round towards the viewer, so
// the shading is a roundness and not a night side.
const vec3 LIGHT_FROM = vec3(-0.45, 0.55, 0.70);

// How much light the side turned away from it keeps, against the side turned
// towards it. The far limb is never darker than this, so a flag there still
// reads as that flag.
const float SHADOW = 0.42;

// The air: the colour of the halo around the globe (atmosphere.ts), and how
// much of it is laid over the ground seen edge-on at the limb. That is what
// joins the disc to the halo, rather than a hard edge with a glow beside it.
//
// A raw ShaderMaterial gets none of three's output colour management, so this
// is written to the framebuffer as it stands: read it as the colour on screen
// rather than as a linear one.
const vec3 AIR = vec3(0.30, 0.62, 1.0);
const float AIR_AT_LIMB = 0.5;

/**
 * How much light reaches a point of the surface, from SHADOW to 1, given its
 * outward normal in view space. Half-Lambert, squared: it wraps the light right
 * round the globe instead of stopping dead at a terminator, and the square
 * brings the falloff back so it still reads as a sphere.
 */
float lightOf(vec3 normal) {
    float facing = dot(normal, normalize(LIGHT_FROM)) * 0.5 + 0.5;
    return mix(SHADOW, 1.0, facing * facing);
}

/**
 * What to multiply a colour on the surface by.
 *
 * 1 at the middle of the disc, which is the point the camera looks straight at
 * and, zoomed in, the whole screen: a player at work on a country sees its
 * flags exactly as bright as before there was a light. The roundness is for the
 * globe seen whole, where the side towards the light comes out brighter still.
 */
float shadeOf(vec3 normal) {
    return lightOf(normal) / lightOf(vec3(0.0, 0.0, 1.0));
}

/** How much of the air to lay over a point of the surface, given its normal. */
float hazeOf(vec3 normal) {
    float edgeOn = 1.0 - clamp(normal.z, 0.0, 1.0);
    return AIR_AT_LIMB * edgeOn * edgeOn * edgeOn;
}

/**
 * A colour as it looks with the air over it, in the light. The air is lit too,
 * so the haze is a bright rim on the side towards the light and fades on the
 * far side, as the halo around the globe does.
 */
vec3 lit(vec3 colour, float shade, float haze) {
    return mix(colour, AIR, haze) * shade;
}
