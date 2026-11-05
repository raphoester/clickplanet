import * as THREE from 'three';
import {mergeVertices} from "three/addons/utils/BufferGeometryUtils.js";

export const EARTH_RADIUS = 0.999

export function innerSphere(): THREE.BufferGeometry {
    return mergeVertices(new THREE.IcosahedronGeometry(EARTH_RADIUS, 50));
}
