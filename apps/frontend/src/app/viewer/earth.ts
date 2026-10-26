import * as THREE from "three"
import transcoderScript from "three/addons/libs/basis/basis_transcoder.js?url"
import transcoderBinary from "three/addons/libs/basis/basis_transcoder.wasm?url"
import {EARTH_ETC1S_URL, EARTH_UASTC_URL, EARTH_URL} from "./earthAsset.ts"
import {type Graphics} from "./graphics.ts"

import vertexShader from "./shaders/earth/vertex.glsl"
import fragmentShader from "./shaders/earth/fragment.glsl"

const textureLoader = new THREE.TextureLoader()

const COMPRESSED_FORMATS = [
    "WEBGL_compressed_texture_astc",
    "EXT_texture_compression_bptc",
    "WEBGL_compressed_texture_etc",
    "WEBGL_compressed_texture_etc1",
    "WEBGL_compressed_texture_s3tc",
    "WEBGL_compressed_texture_pvrtc",
]

const TRANSCODER = new Map([
    ["basis_transcoder.js", transcoderScript],
    ["basis_transcoder.wasm", transcoderBinary],
])

export function createEarth(graphics: Graphics, sphere: THREE.BufferGeometry, renderer: THREE.WebGLRenderer): THREE.Mesh {
    const url = graphics.uastc ? EARTH_UASTC_URL : graphics.etc1s ? EARTH_ETC1S_URL : undefined
    const compressed = url !== undefined && COMPRESSED_FORMATS.some((name) => renderer.extensions.has(name))
    const map = compressed ? new THREE.Texture() : textureLoader.load(EARTH_URL)
    const material = graphics.earth
        ? new THREE.ShaderMaterial({uniforms: {map: {value: map}}, vertexShader, fragmentShader})
        : new THREE.MeshStandardMaterial({map})

    if (compressed) {
        loadCompressed(url, renderer)
            .catch(() => textureLoader.loadAsync(EARTH_URL))
            .then((loaded) => {
                if (material instanceof THREE.ShaderMaterial) material.uniforms.map.value = loaded
                else material.map = loaded
            })
    }

    return new THREE.Mesh(sphere, material)
}

async function loadCompressed(url: string, renderer: THREE.WebGLRenderer): Promise<THREE.Texture> {
    const {KTX2Loader} = await import("three/addons/loaders/KTX2Loader.js")
    const manager = new THREE.LoadingManager()
    manager.setURLModifier((path) => TRANSCODER.get(path) ?? path)
    const loader = new KTX2Loader(manager).detectSupport(renderer)
    try {
        const [texture] = await Promise.all([loader.loadAsync(url), loader.init()])
        return texture
    } finally {
        loader.dispose()
    }
}
