import {syncBoards} from "./writeBoard.ts"

const {name, manifest} = syncBoards()

console.log(`static/boards/${name} ← map/boards/${name}, ${manifest} and the files it names`)
