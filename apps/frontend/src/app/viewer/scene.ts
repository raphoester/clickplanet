import * as THREE from 'three';
import {innerSphere} from "./sphere.ts";
import {layoutViewport} from "./viewport.ts";

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

    const renderer = new THREE.WebGLRenderer({});
    renderer.setSize(width, height);
    renderer.setClearColor(0x000000);
    container.appendChild(renderer.domElement);

    scene.add(new THREE.AmbientLight(0xffffff, 2));

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
            map: textureLoader.load('/static/earth/earth-4k.jpg'),
        })
    ))
}
