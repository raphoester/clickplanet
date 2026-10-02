#ifdef LIT
#include ../light.glsl;
#else
uniform vec3 colour;
#endif

uniform float power;
uniform float intensity;

varying vec3 vViewNormal;

const vec3 VIEW_DIRECTION = vec3(0.0, 0.0, 1.0);

void main() {
    float facing = clamp(dot(-vViewNormal, VIEW_DIRECTION), 0.0, 1.0);

    // Not 1 - facing: the earth hides the shell's middle, so this way it peaks at the limb.
    float glow = pow(facing, power);

    // Intensity goes in alpha: a colour above 1.0 is clamped before it is blended.
#ifdef LIT
    float shade = shadeOf(vec3(vViewNormal.xy, 0.0));
    gl_FragColor = vec4(AIR, clamp(glow * intensity * shade, 0.0, 1.0));
#else
    gl_FragColor = vec4(colour, clamp(glow * intensity, 0.0, 1.0));
#endif
}
