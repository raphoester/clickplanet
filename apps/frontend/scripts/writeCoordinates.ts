import {createHash} from "node:crypto"
import {copyFileSync, mkdirSync, readdirSync, unlinkSync, writeFileSync} from "node:fs"
import {dirname, join, resolve} from "node:path"
import {fileURLToPath} from "node:url"

import {type Coordinates, encodeCoordinates} from "../src/app/viewer/coordinatesBinary.ts"

const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..")
const repoRoot = resolve(frontendRoot, "..", "..")

export const mapDir = join(repoRoot, "map")

export const staticDir = join(frontendRoot, "static")

const viewerDir = join(frontendRoot, "src", "app", "viewer")
const coordinatesPattern = /^coordinates-[0-9a-f]{8}\.bin$/
const bordersPattern = /^borders-[0-9a-f]{8}\.bin$/

export function writeCoordinatesBinary(coordinates: Coordinates): {
    fileName: string
    byteLength: number
    tiles: number
} {
    const buffer = encodeCoordinates(coordinates)
    const bytes = Buffer.from(buffer)
    const hash = createHash("sha256").update(bytes).digest("hex").slice(0, 8)
    const fileName = `coordinates-${hash}.bin`

    sweep(mapDir, coordinatesPattern, fileName)
    writeFileSync(join(mapDir, fileName), bytes)

    syncCoordinatesFromMap()

    return {fileName, byteLength: bytes.byteLength, tiles: coordinates.length}
}

export function syncCoordinatesFromMap(): string {
    const fileName = sharedBlobName(coordinatesPattern)

    sweep(staticDir, coordinatesPattern, fileName)
    copyFileSync(join(mapDir, fileName), join(staticDir, fileName))
    writeFileSync(join(viewerDir, "coordinatesAsset.ts"), assetModule("COORDINATES_URL", fileName))

    return fileName
}

export function bordersPath(fileName: string): string {
    sweep(mapDir, bordersPattern, fileName)
    return join(mapDir, fileName)
}

export function staticBordersPath(fileName: string): string {
    sweep(staticDir, bordersPattern, fileName)
    return join(staticDir, fileName)
}

export function writeBordersAsset(fileName: string): void {
    writeFileSync(join(viewerDir, "bordersAsset.ts"), assetModule("BORDERS_URL", fileName))
}

function sharedBlobName(pattern: RegExp): string {
    const names = readdirSync(mapDir).filter((entry) => pattern.test(entry))
    if (names.length !== 1) {
        throw new Error(`expected one ${mapDir}/${pattern.source}, found ${names.length}`)
    }
    return names[0]
}

function sweep(directory: string, pattern: RegExp, keep: string): void {
    mkdirSync(directory, {recursive: true})
    for (const entry of readdirSync(directory)) {
        if (pattern.test(entry) && entry !== keep) {
            unlinkSync(join(directory, entry))
        }
    }
}

function assetModule(name: string, fileName: string): string {
    return `export const ${name} = "/static/${fileName}"
`
}
