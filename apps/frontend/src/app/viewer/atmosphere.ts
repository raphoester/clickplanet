import * as THREE from 'three';

import vertexShader from "./shaders/atmosphere/vertex.glsl"
import fragmentShader from "./shaders/atmosphere/fragment.glsl"

const RADIUS = 1.06

const POWER = 1.8
const INTENSITY = 3.0

export type Halo = "calm" | "war"

const COLOURS: Record<Halo, THREE.Color> = {
    calm: new THREE.Color(0.30, 0.62, 1.0),
    war: new THREE.Color(1.0, 0.36, 0.12),
}

export function createAtmosphere(lit: boolean): THREE.Mesh {
    const mesh = new THREE.Mesh(
        new THREE.IcosahedronGeometry(RADIUS, 16),
        new THREE.ShaderMaterial({
            defines: {LIT: lit},
            uniforms: {
                colour: {value: COLOURS.calm.clone()},
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

    mesh.renderOrder = -1

    return mesh
}

export function paintHalo(atmosphere: THREE.Mesh, halo: Halo): void {
    const material = atmosphere.material as THREE.ShaderMaterial
    material.uniforms.colour.value.copy(COLOURS[halo])
}
