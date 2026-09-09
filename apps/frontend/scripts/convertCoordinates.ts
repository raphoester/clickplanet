import {readFileSync} from "node:fs"

import {type Coordinates} from "../src/app/viewer/coordinatesBinary.ts"
import {jsonPath, writeCoordinatesBinary} from "./writeCoordinates.ts"

const coordinates = JSON.parse(readFileSync(jsonPath, "utf8")) as Coordinates

const {fileName, byteLength, tiles} = writeCoordinatesBinary(coordinates)
console.log(`wrote static/${fileName}: ${tiles} tiles, ${byteLength} bytes`)
