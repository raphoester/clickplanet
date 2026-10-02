// Fetches the geometry every map blob is built from, and caches it outside the repo.
//
// Natural Earth 1:50m — public domain, ISO codes in the properties. OpenStreetMap has the same
// borders as admin_level=2 relations, but you would have to run an Overpass query and stitch ways
// into rings yourself, and it is ODbL, so share-alike and attribution come with it.
//
// **The tag is pinned.** These files decide where every tile is and who owns the ground under it,
// so two runs a year apart have to produce the same map; `master` does not promise that. Moving the
// pin is a deliberate act that regenerates the blobs and renumbers tiles — see map/README.md.
import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"

export const NATURAL_EARTH_TAG = "v5.1.2"

const base = `https://raw.githubusercontent.com/nvkelso/natural-earth-vector/${NATURAL_EARTH_TAG}/geojson`

// Cached under node_modules/.cache rather than in the repo: it is 3 MB of someone else's data that
// the pin already makes reproducible, and vendoring it would put a second copy of the borders in
// git beside the blob they produce.
const cacheDir = path.resolve(
    path.dirname(fileURLToPath(import.meta.url)),
    "..", "..", "node_modules", ".cache", "natural-earth", NATURAL_EARTH_TAG,
)

/**
 * Reads one Natural Earth geojson by name, downloading it on the first call.
 *
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
