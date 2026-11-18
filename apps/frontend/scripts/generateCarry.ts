import {createHash} from "node:crypto"
import fs from "node:fs"
import path from "node:path"

import {BOARD_DEFINITIONS, boardDir, readBoard} from "./map/board.mjs"
import {carriesDir, carryNamed, encodeCarry, parentsOf} from "./map/carry.mjs"
import {carryStats} from "./map/carryStats.mjs"

// `npm run map:carry -- earth-1 earth-2`: writes map/carries/carry-<from>-<to>-<hash>.bin, each new tile's parent on the
// old board, and prints what it carries. Then `make map` copies it to the backend, the one reader.

const [fromName, toName] = process.argv.slice(2).filter((arg) => !arg.startsWith("-"))
if (!(fromName in BOARD_DEFINITIONS) || !(toName in BOARD_DEFINITIONS) || fromName === toName) {
    throw new Error(`usage: npm run map:carry -- <from board> <to board>, two of ${Object.keys(BOARD_DEFINITIONS).join(", ")}`)
}

const from = readBoard(boardDir(fromName))
const to = readBoard(boardDir(toName))
const started = performance.now()
const parents = parentsOf(from, to)
const took = performance.now() - started
const bytes = encodeCarry(from.manifest, to.manifest, parents)

const name = `carry-${fromName}-${toName}-${createHash("sha256").update(bytes).digest("hex").slice(0, 8)}.bin`
fs.mkdirSync(carriesDir, {recursive: true})
const before = carryNamed(fromName, toName)
if (before !== undefined && before !== name) fs.rmSync(path.join(carriesDir, before))
fs.writeFileSync(path.join(carriesDir, name), bytes)

const stats = carryStats(from, to, parents)
console.log(`map/carries/${name}: ${bytes.length} bytes, ${to.count} tiles in ${took.toFixed(0)} ms`)
console.log(`carried ${stats.carried}, blank ${stats.blank}; old tiles with no child ${stats.orphans}`)
console.log(`children per parent: mean ${stats.meanChildren.toFixed(3)}, inland ${stats.inlandChildren.toFixed(3)}, ` +
    `histogram ${JSON.stringify(stats.histogram)}`)
console.log(`distance to the parent, in the old board's spacings: median ${stats.median.toFixed(3)}, ` +
    `99.9% ${stats.p999.toFixed(3)}, most ${stats.most.toFixed(3)}`)
console.log(`blank tiles whose nearest old tile is within 0.6 spacings: ${stats.blankNear}`)
