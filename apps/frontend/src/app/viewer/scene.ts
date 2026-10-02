import * as THREE from 'three';
import {createAtmosphere} from "./atmosphere.ts";
import {layoutViewport} from "./viewport.ts";
import {createEarth} from "./earth.ts";
import {type Graphics} from "./graphics.ts";

export function setupScene(container: HTMLElement, graphics: Graphics) {
    const scene = new THREE.Scene();
    const cameraSize = 1;
    const {width, height} = layoutViewport();
    const aspect = width / height;
    const camera = new THREE.OrthographicCamera(
        -cameraSize * aspect, cameraSize * aspect,
        cameraSize, -cameraSize, 0.01, 100
    );

    camera.position.z = 5

    const renderer = new THREE.WebGLRenderer({antialias: graphics.antialias});
    renderer.setPixelRatio(pixelRatio(graphics));
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

const MAX_PIXEL_RATIO = 2;

export function pixelRatio(graphics: Graphics): number {
    if (!graphics.ratio) return 1;
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
    graphics: Graphics,
) {
    scene.add(displayPoints);
    scene.add(createEarth(graphics.earth));
    if (!graphics.earth) scene.add(new THREE.AmbientLight(0xffffff, 2));
    scene.add(createAtmosphere(graphics.halo));
}
