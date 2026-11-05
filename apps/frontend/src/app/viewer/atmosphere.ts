import * as THREE from 'three';

import vertexShader from "./shaders/air/vertex.glsl"
import fragmentShader from "./shaders/air/fragment.glsl"

export type Halo = "calm" | "war" | "haunt"

const COLOURS: Record<Halo, THREE.Color> = {
    calm: new THREE.Color(0.30, 0.55, 1.0),
    war: new THREE.Color(1.0, 0.36, 0.12),
    haunt: new THREE.Color(0.62, 0.30, 1.0),
}

export const SKY = 0x080b1a

// After the ground and its outline, before the marks drawn over them.
const AFTER_THE_GROUND = 1.5

// The light and the air, laid over the whole frame from where the globe's disc is on screen.
export function createAtmosphere(): THREE.Mesh {
    const centre = new THREE.Vector2()
    const material = new THREE.ShaderMaterial({
        uniforms: {
            colour: {value: COLOURS.calm.clone()},
            // A raw ShaderMaterial's colours are screen colours, not linear.
            sky: {value: new THREE.Color(SKY).convertLinearToSRGB()},
            centre: {value: centre},
            radius: {value: 1},
        },
        vertexShader,
        fragmentShader,
        transparent: true,
        depthTest: false,
        depthWrite: false,
        blending: THREE.CustomBlending,
        blendEquation: THREE.AddEquation,
        blendSrc: THREE.OneFactor,
        blendDst: THREE.SrcAlphaFactor,
    })

    const geometry = new THREE.BufferGeometry()
    geometry.setAttribute("position", new THREE.BufferAttribute(new Float32Array([-1, -1, 0, 3, -1, 0, -1, 3, 0]), 3))

    const mesh = new THREE.Mesh(geometry, material)
    mesh.frustumCulled = false
    mesh.renderOrder = AFTER_THE_GROUND

    const origin = new THREE.Vector3()
    const size = new THREE.Vector2()
    mesh.onBeforeRender = (renderer, _scene, camera) => {
        renderer.getDrawingBufferSize(size)
        origin.set(0, 0, 0).project(camera)
        centre.set((origin.x + 1) / 2 * size.x, (origin.y + 1) / 2 * size.y)
        material.uniforms.radius.value = camera.projectionMatrix.elements[5] * size.y / 2
    }

    return mesh
}

export function paintHalo(atmosphere: THREE.Mesh, halo: Halo): void {
    const material = atmosphere.material as THREE.ShaderMaterial
    material.uniforms.colour.value.copy(COLOURS[halo])
}
