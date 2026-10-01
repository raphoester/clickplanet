import * as THREE from "three"
import {innerSphere} from "./sphere.ts"
import {EARTH_URL} from "./earthAsset.ts"

import vertexShader from "./shaders/earth/vertex.glsl"
import fragmentShader from "./shaders/earth/fragment.glsl"

const textureLoader = new THREE.TextureLoader()

/**
 * The opaque earth under the tiles, in the same light as they are (see
 * shaders/light.glsl), with a glint on the sea.
 *
 * A raw ShaderMaterial rather than a lit three material: the tiles are drawn by
 * one too, and the only way the ground and the flag over it are shaded alike is
 * for both to call the same function.
 */
export function createEarth(): THREE.Mesh {
    return new THREE.Mesh(
        innerSphere(),
        new THREE.ShaderMaterial({
            uniforms: {
                map: {value: textureLoader.load(EARTH_URL)},
            },
            vertexShader,
            fragmentShader,
        }),
    )
}
