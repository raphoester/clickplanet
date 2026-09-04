/**
 * One-off conversion path: turns the committed `static/coordinates.json` into
 * the binary blob the viewer fetches, without re-running the (slow, map-image
 * dependent) generator. Use it whenever the JSON is the only source of truth
 * for the current map.
 *
 *   npm run coordinates:convert
 */
import {readFileSync} from "node:fs"

import {type Coordinates} from "../src/app/viewer/coordinatesBinary.ts"
import {jsonPath, writeCoordinatesBinary} from "./writeCoordinates.ts"

const coordinates = JSON.parse(readFileSync(jsonPath, "utf8")) as Coordinates

const {fileName, byteLength, tiles} = writeCoordinatesBinary(coordinates)
console.log(`wrote static/${fileName}: ${tiles} tiles, ${byteLength} bytes`)
