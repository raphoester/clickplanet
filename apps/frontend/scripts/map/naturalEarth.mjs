import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"

// Pinned: moving it regenerates the map blobs and renumbers every tile.
export const NATURAL_EARTH_TAG = "v5.1.2"

const base = `https://raw.githubusercontent.com/nvkelso/natural-earth-vector/${NATURAL_EARTH_TAG}/geojson`

const cacheDir = path.resolve(
    path.dirname(fileURLToPath(import.meta.url)),
    "..", "..", "node_modules", ".cache", "natural-earth", NATURAL_EARTH_TAG,
)

/**
 * @param {string} name for example `ne_50m_admin_0_countries`
 * @returns {Promise<{features: object[]}>}
 */
export async function naturalEarth(name) {
    const file = path.join(cacheDir, `${name}.geojson`)
    if (!fs.existsSync(file)) {
        const response = await fetch(`${base}/${name}.geojson`)
        if (!response.ok) {
            throw new Error(`${name} at ${NATURAL_EARTH_TAG}: ${response.status} ${response.statusText}`)
        }
        const body = Buffer.from(await response.arrayBuffer())
        fs.mkdirSync(cacheDir, {recursive: true})
        fs.writeFileSync(file, body)
    }
    return JSON.parse(fs.readFileSync(file, "utf8"))
}
