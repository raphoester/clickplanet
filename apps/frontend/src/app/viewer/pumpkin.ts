import * as THREE from "three"

import vertexShader from "./shaders/pumpkin/vertex.glsl"
import fragmentShader from "./shaders/pumpkin/fragment.glsl"
import {drawQuestionMark} from "./questionMark.ts"

const RADIUS = 0.62
const LOBES = 8
const GROOVE = 0.1
const SQUASH = 0.78
const DENT = 0.16

// Radians a second about x, y and z: a tumble, as the box's.
const SPIN: readonly [number, number, number] = [0.8, 1.1, 0.5]

const CARVING_WIDTH = 512
const CARVING_HEIGHT = 256
const MARK_SIZE = 110

export type Pumpkin = {
    readonly object: THREE.Object3D
    // What a pointer has to hit: the stem is too thin to aim at.
    readonly body: THREE.Mesh
    setOpacity(opacity: number): void
    pose(age: number): void
    dispose(): void
}

export function createPumpkin(): Pumpkin {
    const carving = carvingTexture()

    const bodyGeometry = pumpkinGeometry()
    const bodyMaterial = pumpkinMaterial("#ff8a1f", 1, carving)
    const body = new THREE.Mesh(bodyGeometry, bodyMaterial)

    const stemGeometry = new THREE.CylinderGeometry(0.055, 0.085, 0.3, 10)
    stemGeometry.translate(0, 0.15, 0)
    const stemMaterial = pumpkinMaterial("#5e8f2b", 0, carving)
    const stem = new THREE.Mesh(stemGeometry, stemMaterial)
    stem.position.y = RADIUS * SQUASH - DENT * RADIUS * 0.9
    stem.rotation.z = -0.3

    const object = new THREE.Group()
    // A group starts its own sort: at 0 it would be drawn before the globe's light, and the air laid over it.
    object.renderOrder = 1
    object.add(body, stem)

    const materials = [bodyMaterial, stemMaterial]

    return {
        object,
        body,
        setOpacity: (opacity) => {
            for (const material of materials) material.uniforms.opacity.value = opacity
        },
        pose: (age) => {
            object.rotation.set(age * SPIN[0], age * SPIN[1], age * SPIN[2])
            bodyMaterial.uniforms.flicker.value = flickerAt(age)
        },
        dispose: () => {
            bodyGeometry.dispose()
            stemGeometry.dispose()
            for (const material of materials) material.dispose()
            carving.dispose()
        },
    }
}

// A sphere with LOBES ribs, squashed, dented where the stem grows. Its face, at u = 0.5, looks down +z; the box's
// question marks are carved at u = 0.25, 0.75 and 0, its sides and its back.
export function pumpkinGeometry(): THREE.BufferGeometry {
    const geometry = new THREE.SphereGeometry(RADIUS, 64, 40)
    const position = geometry.attributes.position
    const uv = geometry.attributes.uv
    for (let i = 0; i < position.count; i++) {
        const around = uv.getX(i) * 2 * Math.PI
        const lobe = Math.sqrt(Math.abs(Math.sin(LOBES * around / 2)))
        const ribbed = 1 - GROOVE * (1 - lobe)
        const y = position.getY(i) * SQUASH
        const pole = Math.pow(Math.abs(2 * uv.getY(i) - 1), 8)
        position.setXYZ(i, position.getX(i) * ribbed, y - Math.sign(y) * DENT * RADIUS * pole, position.getZ(i) * ribbed)
    }
    geometry.rotateY(-Math.PI / 2)
    geometry.computeVertexNormals()
    return geometry
}

export function flickerAt(seconds: number): number {
    return 0.88 + 0.12 * Math.sin(seconds * 13) * Math.sin(seconds * 7.3 + 1)
}

function pumpkinMaterial(skin: string, carved: number, carving: THREE.Texture): THREE.ShaderMaterial {
    // A raw ShaderMaterial's colours are screen colours, not linear.
    const screen = (hex: string) => new THREE.Color(hex).convertLinearToSRGB()
    return new THREE.ShaderMaterial({
        uniforms: {
            skin: {value: screen(skin)},
            flesh: {value: screen("#ff9a2e")},
            candle: {value: screen("#ffe25c")},
            flicker: {value: 1},
            carved: {value: carved},
            opacity: {value: 1},
            carving: {value: carving},
        },
        vertexShader,
        fragmentShader,
        transparent: true,
    })
}

// What is carved, white on black, blurred so the shader finds the cut, the shadow round it and the flesh inside.
// Across, 512 pixels go round the pumpkin; down, 256 go pole to pole: the same degrees per pixel both ways.
function carvingTexture(): THREE.CanvasTexture {
    const canvas = document.createElement("canvas")
    canvas.width = CARVING_WIDTH
    canvas.height = CARVING_HEIGHT

    const context = canvas.getContext("2d")
    if (!context) throw new Error("failed to get a 2d context for the pumpkin's carving")

    context.fillStyle = "#000000"
    context.fillRect(0, 0, CARVING_WIDTH, CARVING_HEIGHT)
    context.filter = "blur(3px)"
    context.fillStyle = "#ffffff"

    const shape = (points: readonly [number, number][]) => {
        context.beginPath()
        context.moveTo(...points[0])
        for (const point of points.slice(1)) context.lineTo(...point)
        context.closePath()
        context.fill()
    }
    shape([[204, 116], [226, 80], [246, 116]])
    shape([[266, 116], [286, 80], [308, 116]])
    shape([[247, 134], [256, 118], [265, 134]])
    shape([
        [192, 138], [212, 150], [224, 146], [232, 156], [244, 151], [256, 158], [268, 151], [280, 156], [288, 146],
        [300, 150], [320, 138], [310, 162], [292, 176], [282, 168], [270, 180], [256, 182], [242, 180], [230, 168],
        [220, 176], [202, 162],
    ])

    // The back's mark straddles the seam, so it is drawn at both ends.
    for (const across of [0, 0.25, 0.75, 1]) {
        drawQuestionMark(context, across * CARVING_WIDTH - MARK_SIZE / 2, CARVING_HEIGHT / 2 - MARK_SIZE / 2 - 4, MARK_SIZE, "#ffffff", "#000000")
    }

    const texture = new THREE.CanvasTexture(canvas)
    texture.colorSpace = THREE.NoColorSpace
    return texture
}
