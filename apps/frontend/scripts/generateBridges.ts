import {boardAsked, boardDir, readBoard, readCapitals} from "./map/board.mjs"
import {bridgesOf, lanesOf, piecesApart, snapKmOf} from "./map/bridges.mjs"
import {tileLattice} from "./map/tileLattice.mjs"
import {syncBoards, writeBoardJSON, writeManifest} from "./writeBoard.ts"

// `npm run bridges -- earth-2`; earth-1 when no board is named.
const board = readBoard(boardDir(boardAsked(process.argv.slice(2)).name))
const {capitals} = readCapitals(board)
const lattice = tileLattice(board)

const {bridges, pieces, stones, lanes, seconds} = bridgesOf({positions: board.positions, ...lattice, capitals}, lanesOf(board.manifest.name), snapKmOf(board.manifest.spacing))
const apart = piecesApart(lattice, board.count, bridges)
if (apart > 0) throw new Error(`${apart} pieces of land no bridge reaches`)

const file = writeBoardJSON(board.manifest.name, "bridges", {tiles: board.count, bridges})
const manifest = writeManifest(board.manifest.name)
syncBoards()
console.log(`wrote map/boards/${board.manifest.name}/${file.fileName} and ${manifest}: ${bridges.length} bridges over`
    + ` ${pieces} pieces of land, ${stones} stepping stones, ${lanes} sea lanes and ${seconds} second ways
next:
  cd ../backend && make map   # copy it, then commit every copy`)
