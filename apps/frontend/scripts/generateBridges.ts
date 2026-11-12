import {readCapitals, readCoordinates} from "./map/blob.mjs"
import {bridgesOf, piecesApart} from "./map/bridges.mjs"
import {tileLattice} from "./map/tileLattice.mjs"
import {writeSharedJSON} from "./writeCoordinates.ts"

const {name, count, positions} = readCoordinates()
const {capitals} = readCapitals()
const lattice = tileLattice(positions)

const {bridges, pieces, stones, lanes, seconds} = bridgesOf({positions, ...lattice, capitals})
const apart = piecesApart(lattice, count, bridges)
if (apart > 0) throw new Error(`${apart} pieces of land no bridge reaches`)

const file = writeSharedJSON("bridges", {tiles: count, bridges})
console.log(`wrote map/${file.fileName} from ${name}: ${bridges.length} bridges over ${pieces} pieces of land,`
    + ` ${stones} stepping stones, ${lanes} sea lanes and ${seconds} second ways
next:
  cd ../backend && make map   # copy it, then commit every copy`)
