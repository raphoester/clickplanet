import * as THREE from "three";
import {integerToColor} from "./pickingColors.ts";
import {IUniform} from "three";

// @ts-expect-error typescript does not know about vite-plugin-glsl
import displayVertex from "./shaders/display/vertex.glsl"
// @ts-expect-error typescript does not know about vite-plugin-glsl
import displayFragment from "./shaders/display/fragment.glsl"

// @ts-expect-error typescript does not know about vite-plugin-glsl
import pickerVertex from "./shaders/picker/vertex.glsl"
// @ts-expect-error typescript does not know about vite-plugin-glsl
import pickerFragment from "./shaders/picker/fragment.glsl"

import {COORDINATES_URL} from "./coordinatesAsset.ts"
import {decodeCoordinates, type PointGeometryData} from "./coordinatesBinary.ts"

export type {PointGeometryData}

/**
 * Fetches the tile coordinates blob and views it as the typed arrays the
 * geometry needs. The data used to be a build-time JSON import, which put 25 MB
 * into the main chunk; fetching the binary keeps the chunk small and skips both
 * the JSON parse and the element-by-element copy into Float32Array.
 */
export async function loadPointGeometryData(signal?: AbortSignal): Promise<PointGeometryData> {
    const response = await fetch(COORDINATES_URL, {signal})
    if (!response.ok) {
        throw new Error(`Failed to fetch ${COORDINATES_URL}: ${response.status} ${response.statusText}`)
    }
    return decodeCoordinates(await response.arrayBuffer())
}

export function createPoints(uniforms: { [uniform: string]: IUniform; }, data: PointGeometryData) {
    const {positions, uvs, size} = data;

    const colors = generatePickingColors(size);
    const pickerGeometry = new THREE.BufferGeometry();
    const displayGeometry = new THREE.BufferGeometry();

    pickerGeometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    pickerGeometry.setAttribute('uv', new THREE.BufferAttribute(uvs, 2));
    pickerGeometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));

    const hoveredArr = new Float32Array(size).fill(0);
    displayGeometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
    displayGeometry.setAttribute('uv', new THREE.BufferAttribute(uvs, 2));
    displayGeometry.setAttribute('hover', new THREE.BufferAttribute(hoveredArr, 1));

    const displayPoints = new THREE.Points(displayGeometry, new THREE.ShaderMaterial({
        transparent: true,
        uniforms,
        vertexShader: displayVertex,
        fragmentShader: displayFragment,
    }));

    const pickingPoints = new THREE.Points(pickerGeometry, new THREE.ShaderMaterial({
        uniforms,
        vertexShader: pickerVertex,
        fragmentShader: pickerFragment,
    }));

    return {
        displayPoints,
        pickingPoints,
        size
    };
}

function generatePickingColors(size: number): Float32Array {
    const colors = new Float32Array(size * 3);
    for (let i = 0; i < size; i++) {
        const [r, g, b] = integerToColor(i + 1);
        colors[i * 3] = r / 255;
        colors[i * 3 + 1] = g / 255;
        colors[i * 3 + 2] = b / 255;
    }
    return colors;
}