import {syncBoards} from "./writeBoard.ts"

for (const {name, manifest} of syncBoards()) {
    console.log(`static/boards/${name} ← map/boards/${name}, ${manifest} and the files it names`)
}
