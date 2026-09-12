import {syncCoordinatesFromMap} from "./writeCoordinates.ts"

// `npm run map` — this app's half of the shared /map blob. The backend's half is its `make map`.
const fileName = syncCoordinatesFromMap()

console.log(`static/${fileName} ← map/${fileName}`)
