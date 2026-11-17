import * as THREE from "three"
import {OPEN_SEA_COLOUR} from "./terrain.ts"

import vertexShader from "./shaders/earth/vertex.glsl"
import fragmentShader from "./shaders/earth/fragment.glsl"

// The open sea: the land and the coast are the territory layer's, drawn over it.
export function createEarth(glint: boolean, sphere: THREE.BufferGeometry): THREE.Mesh {
    return new THREE.Mesh(
        sphere,
        new THREE.ShaderMaterial({
            defines: {SEA_GLINT: glint},
            uniforms: {
                sea: {value: OPEN_SEA_COLOUR},
            },
            vertexShader,
            fragmentShader,
        }),
    )
}
