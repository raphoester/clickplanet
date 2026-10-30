import * as THREE from "three"
import {EARTH_URL} from "./earthAsset.ts"

import vertexShader from "./shaders/earth/vertex.glsl"
import fragmentShader from "./shaders/earth/fragment.glsl"

const textureLoader = new THREE.TextureLoader()

const ANISOTROPY = 8

export function createEarth(glint: boolean, sphere: THREE.BufferGeometry): THREE.Mesh {
    const map = textureLoader.load(EARTH_URL)
    map.anisotropy = ANISOTROPY

    return new THREE.Mesh(
        sphere,
        new THREE.ShaderMaterial({
            defines: {SEA_GLINT: glint},
            uniforms: {
                map: {value: map},
            },
            vertexShader,
            fragmentShader,
        }),
    )
}
