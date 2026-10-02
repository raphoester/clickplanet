import * as THREE from 'three';

import vertexShader from "./shaders/atmosphere/vertex.glsl"
import fragmentShader from "./shaders/atmosphere/fragment.glsl"

const RADIUS = 1.06

const POWER = 1.8
const INTENSITY = 3.0

const COLOUR = new THREE.Color(0.30, 0.62, 1.0)

export function createAtmosphere(lit: boolean): THREE.Mesh {
    const mesh = new THREE.Mesh(
        new THREE.IcosahedronGeometry(RADIUS, 16),
        new THREE.ShaderMaterial({
            defines: {LIT: lit},
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

    // Before the tiles: they write depth, and one past the limb would notch the halo.
    mesh.renderOrder = -1

    return mesh
}
