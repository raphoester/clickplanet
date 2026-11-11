import * as THREE from "three"
import {LATTICE_COLS, latticeFaces, type TileCells} from "../../domain/latticeCells.ts"
import {EARTH_RADIUS} from "./sphere.ts"

import vertexShader from "./shaders/territory/vertex.glsl"
import fragmentShader from "./shaders/territory/fragment.glsl"

// Must match CELL_COLUMNS in shaders/territory/fragment.glsl.
const CELL_COLUMNS = 256

// Just outside the tiles' sphere, so every pixel of it is under the layer's mesh: the shader finds the sphere itself.
const CARRIER_RADIUS = 1.002

const FACE_TEXELS = 8

// Drawn after the earth and before the tiles, which fade in over it.
const BEFORE_THE_TILES = -1

export type TerritoryLayerInputs = {
    sphere: THREE.BufferGeometry
    cells: TileCells
    tileData: THREE.Texture
    flagRectangles: THREE.Texture
    uniforms: {[uniform: string]: THREE.IUniform}
}

// The flags painted on the ground zoomed out: each pixel finds its tile's cell and wears its holder's flag.
export function createTerritoryLayer({sphere, cells, tileData, flagRectangles, uniforms}: TerritoryLayerInputs): THREE.Mesh {
    const faces = latticeFaces()

    const faceData = new Float32Array(FACE_TEXELS * faces.length * 4)
    faces.forEach(({origin, toB, toC, alongOf, downOf, ids}, face) => {
        const texels = [
            [...origin, 0], [...toB, 0], [...toC, 0], [...alongOf, 0], [...downOf, 0],
            [...ids.corners, ids.inside],
            [...ids.edges.map(({at}) => at), 0],
            [...ids.edges.map(({by}) => by), 0],
        ]
        texels.forEach((texel, column) => faceData.set(texel, (face * FACE_TEXELS + column) * 4))
    })
    const faceTexture = new THREE.DataTexture(faceData, FACE_TEXELS, faces.length, THREE.RGBAFormat, THREE.FloatType)
    faceTexture.needsUpdate = true

    const material = new THREE.ShaderMaterial({
        uniforms: {
            ...uniforms,
            faceNormals: {value: faces.map(({normal, height}) => new THREE.Vector4(...normal, height))},
            latticeFaces: {value: faceTexture},
            latticeCols: {value: LATTICE_COLS},
            tileCells: {value: cellTexture(cells.words)},
            tileData: {value: tileData},
            flagRectangles: {value: flagRectangles},
        },
        vertexShader,
        fragmentShader,
        transparent: true,
        depthWrite: false,
    })

    const mesh = new THREE.Mesh(sphere, material)
    mesh.scale.setScalar(CARRIER_RADIUS / EARTH_RADIUS)
    mesh.renderOrder = BEFORE_THE_TILES
    return mesh
}

function cellTexture(words: Uint32Array): THREE.DataTexture {
    const pairs = words.length / 2
    const rows = Math.ceil(pairs / CELL_COLUMNS)
    const data = new Uint32Array(CELL_COLUMNS * rows * 2)
    data.set(words)
    const texture = new THREE.DataTexture(data, CELL_COLUMNS, rows, THREE.RGIntegerFormat, THREE.UnsignedIntType)
    texture.internalFormat = "RG32UI"
    texture.needsUpdate = true
    return texture
}
