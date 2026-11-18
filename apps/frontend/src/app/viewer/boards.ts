import {type BoardAsset, FIRST_BOARD} from "../../domain/board.ts"
import {BOARDS} from "./boardsAsset.ts"

export {FIRST_BOARD}

// A board this build does not carry: the server plays a board added after the bundle was built.
export class UnknownBoardError extends Error {
    constructor(readonly board: string) {
        super(`this page carries no board "${board}": it has ${Object.keys(BOARDS).join(", ")}`)
        this.name = "UnknownBoardError"
    }
}

export function boardNamed(name: string): BoardAsset | undefined {
    return Object.hasOwn(BOARDS, name) ? BOARDS[name] : undefined
}

export function boardOf(name: string): BoardAsset {
    const board = boardNamed(name)
    if (board === undefined) throw new UnknownBoardError(name)
    return board
}
