import * as THREE from 'three';
import {createAtmosphere} from "./atmosphere.ts";
import {layoutViewport} from "./viewport.ts";
import {createEarth} from "./earth.ts";

export function setupScene(container: HTMLElement) {
    const scene = new THREE.Scene();
    const cameraSize = 1;
    const {width, height} = layoutViewport();
    const aspect = width / height;
    const camera = new THREE.OrthographicCamera(
        -cameraSize * aspect, cameraSize * aspect,
        cameraSize, -cameraSize, 0.01, 100
    );

    camera.position.z = 5

    // No `preserveDrawingBuffer`. It would make the driver keep a second copy of
    // the buffer for every frame of every session — a permanent cost on the
    // thing this page is, a globe at 60fps — to serve a share button that is
    // pressed once in a while, if at all. The share capture reads the buffer
    // from inside the render loop instead, while it is still there; see
    // `readDrawingBuffer` in capture.ts.
    //
    // No `antialias` either. Turned on for screens below a ratio of 2, it
    // brightened the whole canvas about 2.5 times on Windows Chrome with an
    // Intel GPU (ANGLE on Direct3D 11): the planet came out nearly white, the
    // unlit grey outlines included, so it was the multisampled buffer and not
    // anything drawn into it. A smooth limb is not worth that.
    const renderer = new THREE.WebGLRenderer({});
    renderer.setPixelRatio(pixelRatio());
    renderer.setSize(width, height);
    renderer.setClearColor(0x000000);
    container.appendChild(renderer.domElement);

    const cleanup = () => {
        renderer.setAnimationLoop(null);
        disposeScene(scene);
        renderer.domElement.remove();
        renderer.dispose();
        renderer.forceContextLoss();
    }

    return {scene, camera, cameraSize, renderer, cleanup};
}

/**
 * The most drawing-buffer pixels drawn per CSS pixel.
 *
 * The canvas used to be drawn at one, so a phone or a laptop at two or three
 * stretched every frame to two or three times its size and the whole globe came
 * out soft. Two is where it stops: a phone at three would draw 2.25 times the
 * pixels again, for a difference nobody sees at arm's length.
 */
const MAX_PIXEL_RATIO = 2;

/**
 * How many drawing-buffer pixels the canvas has per CSS pixel on this screen.
 *
 * Every size in this viewer written in pixels — a tile, a line, a mark — is a
 * CSS pixel, and is multiplied by this on its way to the GPU, so a screen with
 * more pixels draws the same picture sharper rather than a smaller one.
 */
export function pixelRatio(): number {
    return Math.min(window.devicePixelRatio || 1, MAX_PIXEL_RATIO);
}

export function disposeScene(scene: THREE.Scene) {
    scene.traverse((object) => {
        const {geometry, material} = object as Partial<THREE.Mesh>;
        geometry?.dispose();
        for (const single of Array.isArray(material) ? material : material ? [material] : []) {
            disposeMaterial(single);
        }
    });
    scene.clear();
}

export function disposeMaterial(material: THREE.Material) {
    for (const value of Object.values(material)) {
        if (value instanceof THREE.Texture) value.dispose();
    }
    if (material instanceof THREE.ShaderMaterial) {
        for (const uniform of Object.values(material.uniforms)) {
            if (uniform.value instanceof THREE.Texture) uniform.value.dispose();
        }
    }
    material.dispose();
}

export function addDisplayObjects(
    scene: THREE.Scene,
    displayPoints: THREE.Points,
) {
    scene.add(displayPoints);
    scene.add(createEarth());
    scene.add(createAtmosphere());
}
