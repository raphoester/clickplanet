import * as THREE from "three"
import {EARTH_URL} from "./earthAsset.ts"

import vertexShader from "./shaders/earth/vertex.glsl"
import fragmentShader from "./shaders/earth/fragment.glsl"

const textureLoader = new THREE.TextureLoader()

const ANISOTROPY = 8

export function createEarth(lit: boolean, sphere: THREE.BufferGeometry): THREE.Mesh {
    const map = textureLoader.load(EARTH_URL)
    map.anisotropy = ANISOTROPY
    if (!lit) {
        // Decoded and encoded again on the way out, so each pixel is the photo's own colour.
        map.colorSpace = THREE.SRGBColorSpace
        return new THREE.Mesh(sphere, new THREE.MeshBasicMaterial({map}))
    }

    return new THREE.Mesh(
        sphere,
        new THREE.ShaderMaterial({
            uniforms: {
                map: {value: map},
            },
            vertexShader,
            fragmentShader,
        }),
    )
}
