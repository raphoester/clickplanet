import {createHash} from "node:crypto"
import {readdirSync, readFileSync, unlinkSync, writeFileSync} from "node:fs"

import {decodeCoordinates} from "../src/app/viewer/coordinatesBinary.ts"
import {LATTICE_DETAIL, positionsHash} from "../src/domain/lattice.ts"
import {encodeLandmask, landmaskOf, rebuildTilePositions} from "../src/domain/landmask.ts"

const coordinatesName = only(/^coordinates-[0-9a-f]{8}\.bin$/)
const coordinates = readFileSync(`static/${coordinatesName}`)
const {positions} = decodeCoordinates(coordinates.buffer.slice(coordinates.byteOffset, coordinates.byteOffset + coordinates.byteLength))

const landmask = landmaskOf(LATTICE_DETAIL, positions)
const bytes = Buffer.from(encodeLandmask(landmask))
if (positionsHash(rebuildTilePositions(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength))) !== landmask.hash) {
    throw new Error("the landmask does not rebuild the positions it was made from")
}

const fileName = `landmask-${createHash("sha256").update(bytes).digest("hex").slice(0, 8)}.bin`
for (const entry of readdirSync("static")) {
    if (/^landmask-[0-9a-f]{8}\.bin$/.test(entry) && entry !== fileName) unlinkSync(`static/${entry}`)
}
writeFileSync(`static/${fileName}`, bytes)
writeFileSync("src/app/viewer/landmaskAsset.ts", `export const LANDMASK_URL = "/static/${fileName}"
`)
console.log(`wrote static/${fileName}: ${landmask.tiles} tiles of ${coordinatesName}, ${bytes.byteLength} bytes`)

function only(pattern: RegExp): string {
    const found = readdirSync("static").filter((entry) => pattern.test(entry))
    if (found.length !== 1) throw new Error(`expected one static/${pattern.source}, found ${found.length}`)
    return found[0]
}
