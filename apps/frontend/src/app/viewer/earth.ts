import * as THREE from "three"
import {innerSphere} from "./sphere.ts"
import {EARTH_URL} from "./earthAsset.ts"

import vertexShader from "./shaders/earth/vertex.glsl"
import fragmentShader from "./shaders/earth/fragment.glsl"

const textureLoader = new THREE.TextureLoader()

export function createEarth(lit: boolean): THREE.Mesh {
    const map = textureLoader.load(EARTH_URL)
    if (!lit) return new THREE.Mesh(innerSphere(), new THREE.MeshStandardMaterial({map}))

    return new THREE.Mesh(
        innerSphere(),
        new THREE.ShaderMaterial({
            uniforms: {
                map: {value: map},
            },
            vertexShader,
            fragmentShader,
        }),
    )
}
