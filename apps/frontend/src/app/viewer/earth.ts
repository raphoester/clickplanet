import * as THREE from "three"
import {innerSphere} from "./sphere.ts"
import {EARTH_URL} from "./earthAsset.ts"

import vertexShader from "./shaders/earth/vertex.glsl"
import fragmentShader from "./shaders/earth/fragment.glsl"

const textureLoader = new THREE.TextureLoader()

/**
 * The opaque earth under the tiles.
 *
 * `lit`, it is in the same light as they are (see shaders/light.glsl), with a
 * glint on the sea: a raw ShaderMaterial rather than a lit three material,
 * because the tiles are drawn by one too, and the only way the ground and the
 * flag over it are shaded alike is for both to call the same function.
 *
 * Otherwise it is three's standard material, as it always was, and the scene
 * has to light it (`addDisplayObjects`). See graphics.ts for why both are here.
 */
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
