import {writeFileSync} from "node:fs";

import sharp from "sharp";
import * as THREE from "three";

import {jsonPath, writeCoordinatesBinary} from "./writeCoordinates.ts";

const args = process.argv.slice(2);
if (args.length < 2) {
    console.error('Usage: node computePoints.js <detail> <mapFilePath> [threshold]');
    process.exit(1);
}

const detail = parseInt(args[0]);
const mapFilePath = args[1];
const threshold = args[2] ? parseInt(args[2]) : 128;

loadMap().then(({pixels, width, height, channels}) => {
    const baseCoordinates = generateBaseCoordinates(detail);
    const landCoordinates = filterOutCoordinatesNotOnLand(baseCoordinates, pixels, width, height, channels);

    // The JSON stays the human-readable generator output kept in the repo; only
    // the binary is shipped, so both are written from the same run to keep them
    // describing the same map.
    writeFileSync(jsonPath, JSON.stringify(landCoordinates));
    console.log(`wrote static/coordinates.json: ${landCoordinates.length} tiles`);

    const {fileName, byteLength, tiles} = writeCoordinatesBinary(landCoordinates);
    console.log(`wrote static/${fileName}: ${tiles} tiles, ${byteLength} bytes`);
})

type Coordinates = {
    positions: number[];
    uvs: number[];
    length: number;
}

async function loadMap(): Promise<{ pixels: Uint8Array; width: number; height: number; channels: number }> {
    const {data, info} = await sharp(mapFilePath).raw().toBuffer({resolveWithObject: true});
    return {pixels: data, width: info.width, height: info.height, channels: info.channels};
}

function generateBaseCoordinates(detail: number): Coordinates {
    const geometry = new THREE.IcosahedronGeometry(1, detail);
    const pos = geometry.attributes.position.array;
    const uvs = geometry.attributes.uv.array;

    const retPositions: number[] = [];
    const retUVs: number[] = [];
    const uniqueVertices = new Map<string, { index: number }>();

    for (let i = 0; i < pos.length; i += 3) {
        const x = pos[i].toFixed(6);
        const y = pos[i + 1].toFixed(6);
        const z = pos[i + 2].toFixed(6);

        const key = `${x},${y},${z}`;
        if (!uniqueVertices.has(key)) {
            uniqueVertices.set(key, {index: retPositions.length / 3});
            retPositions.push(pos[i], pos[i + 1], pos[i + 2]);

            const uvIndex = (i / 3) * 2;
            retUVs.push(uvs[uvIndex], uvs[uvIndex + 1]);
        }
    }

    return {
        positions: retPositions,
        uvs: retUVs,
        length: retPositions.length / 3,
    };
}

function filterOutCoordinatesNotOnLand(coordinates: Coordinates, pixels: Uint8Array, width: number, height: number, channels: number): Coordinates {
    const newPositions = [];
    const newUVs = [];

    for (let i = 0; i < coordinates.positions.length / 3; i++) {
        if (isLand(coordinates.uvs[i * 2], coordinates.uvs[i * 2 + 1], pixels, width, height, channels)) {
            newPositions.push(
                coordinates.positions[i * 3],
                coordinates.positions[i * 3 + 1],
                coordinates.positions[i * 3 + 2]
            );
            newUVs.push(
                coordinates.uvs[i * 2],
                coordinates.uvs[i * 2 + 1]
            );
        }
    }

    coordinates.positions = newPositions;
    coordinates.uvs = newUVs;
    coordinates.length = newPositions.length / 3; // Mettre à jour la longueur

    return {
        positions: Array.from(coordinates.positions),
        uvs: Array.from(coordinates.uvs),
        length: coordinates.length
    }
}

// Samples the map the same way the GPU samples the earth texture, so the dots
// land where the visible continents are.
//
// The globe and the dots are both IcosahedronGeometry, so they share UVs, and
// v = 1 is the north pole. Three.js loads textures with flipY = true, so v = 1
// reads the TOP row of the image — hence height - 1 - (v * height), not
// v * height.
//
// Both coordinates are clamped. u reaches exactly 1.0 on the seam vertices,
// and floor(1.0 * width) is one past the last column: unclamped it silently
// wraps into the first column of the next row and misclassifies those tiles.
function isLand(
    u: number,
    v: number,
    pixels: Uint8Array,
    width: number,
    height: number,
    channels: number,
): boolean {
    const x = Math.min(Math.floor(u * width), width - 1);
    const y = Math.min(Math.max(height - 1 - Math.floor(v * height), 0), height - 1);
    const index = (y * width + x) * channels;
    return pixels[index] < threshold
}