import {createHash} from "node:crypto"
import {readFileSync} from "node:fs"
import {basename} from "node:path"
import {fileURLToPath} from "node:url"
import sharp from "sharp"
import {describe, expect, it} from "vitest"
import countries from "../../../static/countries/countries.json"
import {regions} from "./atlas.ts"
import {ATLAS_SIZE, ATLAS_URL} from "./atlasAsset.ts"
import {decodeBorders} from "./borderField.ts"
import {BORDERS_URL} from "./bordersAsset.ts"

const STATIC = fileURLToPath(new URL("../../../static/", import.meta.url))
const staticFile = (url: string) => readFileSync(STATIC + url.replace(/^\/static\//, ""))

describe("the flag atlas", () => {
    const atlas = staticFile(ATLAS_URL)

    it("is named by its own bytes, so a browser can never pair old pixels with new regions", () => {
        const hash = createHash("sha256").update(atlas).digest("hex").slice(0, 8)
        expect(basename(ATLAS_URL)).toBe(`atlas-${hash}.webp`)
    })

    it("is the size the shader is told", async () => {
        const {width, height} = await sharp(atlas).metadata()
        expect({width, height}).toEqual(ATLAS_SIZE)
    })

    it("holds every region inside it", () => {
        for (const [code, region] of regions) {
            expect(region.x + region.width, code).toBeLessThanOrEqual(ATLAS_SIZE.width)
            expect(region.y + region.height, code).toBeLessThanOrEqual(ATLAS_SIZE.height)
        }
    })

    it("has a flag for every country on the map and in the picker", () => {
        const blob = staticFile(BORDERS_URL)
        const {codes} = decodeBorders(blob.buffer.slice(blob.byteOffset, blob.byteOffset + blob.byteLength))
        const wanted = new Set([...codes.filter((code) => code !== ""), ...Object.keys(countries)])
        const missing = [...wanted].filter((code) => !regions.has(code))
        expect(missing).toEqual([])
    })
})
