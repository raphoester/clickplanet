#include ../light.glsl;

uniform float power;
uniform float intensity;

varying vec3 vViewNormal;

// The camera is orthographic, so every fragment is looked at from the same
// direction and that direction is the view space +Z axis. Deriving it from
// `cameraPosition - worldPosition`, as a perspective scene would, tilts the
// falloff towards wherever the camera happens to sit and leaves the halo
// lopsided.
const vec3 VIEW_DIRECTION = vec3(0.0, 0.0, 1.0);

void main() {
    // Drawn back side, so the fragments are the far hemisphere and their
    // outward normals point away from the viewer. Negated, the term reads 0
    // on the silhouette of this shell and 1 at the middle of its disc.
    float facing = clamp(dot(-vViewNormal, VIEW_DIRECTION), 0.0, 1.0);

    // The Fresnel is taken this way round rather than as `1 - facing` because
    // the earth covers everything but the annulus between its own limb and
    // this shell's silhouette: `1 - facing` is at its brightest exactly where
    // that annulus ends, so the glow would stop dead at its peak and read as
    // an outline. This way it is brightest against the limb and reaches zero
    // out at the edge, which is the halo fading into space. `power` tightens
    // it onto the limb.
    float glow = pow(facing, power);

    // `intensity` scales the alpha and not the colour: additive blending
    // multiplies by the alpha, and the framebuffer holds bytes, so a colour
    // taken above 1.0 is clamped to white before it is ever blended. Folded
    // into the rgb this read as a grey halo with the blue scaled out of it.
    //
    // Lit like the ground under it, which faces out across the screen the same
    // way this fragment's normal does: bright on the side towards the light,
    // faint on the far side, as the haze over the limb is. Left unnormalised,
    // because between the earth's limb and this shell it is never shorter than
    // 0.94, and inside the earth's disc, where it shrinks to nothing, the depth
    // test has already thrown the fragment away.
    float shade = shadeOf(vec3(vViewNormal.xy, 0.0));
    gl_FragColor = vec4(AIR, clamp(glow * intensity * shade, 0.0, 1.0));
}
