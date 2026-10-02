import {syncCoordinatesFromMap} from "./writeCoordinates.ts"

const fileName = syncCoordinatesFromMap()

console.log(`static/${fileName} ← map/${fileName}`)
