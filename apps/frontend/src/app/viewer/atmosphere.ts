import * as THREE from 'three';

import vertexShader from "./shaders/atmosphere/vertex.glsl"
import fragmentShader from "./shaders/atmosphere/fragment.glsl"

// Far enough outside the 1.0 tile shell that the halo is a band of its own
// rather than a haze over the flags, and close enough that it still reads as
// this planet's air. The earth is opaque and drawn first, so everything of
// this shell inside the globe's limb is thrown away by the depth test: what
// survives is the annulus between 0.999 and here.
const RADIUS = 1.06

// Tuned together: `power` pulls the glow onto the limb, `intensity` says how
// much of it there is. The visible annulus only ever reaches a Fresnel term of
// about 0.33, so the intensity is well above 1 to make up for it.
const POWER = 1.8
const INTENSITY = 3.0

// A raw ShaderMaterial gets none of three's output colour management, so this
// is written to the framebuffer as it stands: read it as the halo's colour on
// screen rather than as a linear one.
const COLOUR = new THREE.Color(0.30, 0.62, 1.0)

export function createAtmosphere(): THREE.Mesh {
    const mesh = new THREE.Mesh(
        new THREE.IcosahedronGeometry(RADIUS, 16),
        new THREE.ShaderMaterial({
            uniforms: {
                colour: {value: COLOUR},
                power: {value: POWER},
                intensity: {value: INTENSITY},
            },
            vertexShader,
            fragmentShader,
            side: THREE.BackSide,
            transparent: true,
            blending: THREE.AdditiveBlending,
            depthWrite: false,
        }),
    )

    // The tiles are transparent too, and a transparent material still writes
    // depth: a tile splatted past the limb would punch a notch out of the halo
    // behind it. Ordering this first in the transparent pass means only the
    // opaque earth has written any depth by the time it is drawn, so the band
    // is whole. It writes no depth itself, so the tiles that follow are
    // unaffected either way.
    mesh.renderOrder = -1

    return mesh
}
