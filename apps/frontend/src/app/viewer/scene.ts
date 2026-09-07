import * as THREE from 'three';
import {innerSphere} from "./sphere.ts";


export function setupScene(container: HTMLElement) {
    const scene = new THREE.Scene();
    const cameraSize = 1;
    const aspect = window.innerWidth / window.innerHeight;
    const camera = new THREE.OrthographicCamera(
        -cameraSize * aspect, cameraSize * aspect,
        cameraSize, -cameraSize, 0.01, 100
    );

    camera.position.z = 5

    const renderer = new THREE.WebGLRenderer({});
    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.setClearColor(0x000000);
    container.appendChild(renderer.domElement);

    scene.add(new THREE.AmbientLight(0xffffff, 2));

    /**
     * Removes only this run's canvas, not everything in the container: under
     * <StrictMode> a discarded run and the surviving one share the container,
     * and each renderer owns a WebGL context that has to be released explicitly
     * or the browser drops the oldest context once the limit is reached.
     */
    const cleanup = () => {
        renderer.setAnimationLoop(null);
        disposeScene(scene);
        renderer.domElement.remove();
        renderer.dispose();
        renderer.forceContextLoss();
    }

    return {scene, camera, cameraSize, renderer, cleanup};
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

const textureLoader = new THREE.TextureLoader();

export function addDisplayObjects(
    scene: THREE.Scene,
    displayPoints: THREE.Points,
) {
    scene.add(displayPoints);
    scene.add(new THREE.Mesh(
        innerSphere(),
        new THREE.MeshStandardMaterial({
            // 4096x2048, not the 16200x8100 original. The GPU stores textures
            // uncompressed, so that one cost 501 MB of video memory (667 MB
            // with mipmaps) regardless of being 4.5 MB on disk. iOS Safari
            // kills a tab well below that, which showed up as the page
            // reloading in a loop on iPhone. This is 43 MB.
            map: textureLoader.load('/static/earth/earth-4k.jpg'),
        })
    ))
}