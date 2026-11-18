import {createHash} from "node:crypto"
import {readdirSync, readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {BOARDS} from "./boardsAsset.ts"
import {boardNamed, boardOf, FIRST_BOARD, UnknownBoardError} from "./boards.ts"

const STATIC = fileURLToPath(new URL("../../../static/", import.meta.url))

const hashOf = (bytes: Buffer) => createHash("sha256").update(bytes).digest("hex").slice(0, 8)

describe("the boards' table", () => {
    it("holds every board the page has a copy of, by its name", () => {
        expect(Object.keys(BOARDS).sort()).toEqual(readdirSync(`${STATIC}boards`).sort())
        for (const [name, board] of Object.entries(BOARDS)) expect(board.name).toBe(name)
        expect(BOARDS).toHaveProperty(FIRST_BOARD)
    })

    for (const board of Object.values(BOARDS)) {
        describe(board.name, () => {
            const directory = `${STATIC}boards/${board.name}/`

            it("serves its own files beside its copy, each named by its content", () => {
                for (const [kind, file] of Object.entries(board.page)) {
                    expect(board.urls[kind as keyof typeof board.page]).toBe(`/static/boards/${board.name}/${file}`)
                    expect(file).toMatch(new RegExp(`^${kind}-${hashOf(readFileSync(directory + file))}\\.(bin|json)$`))
                }
            })

            it("names its landmasses by the rows of its own borders file", () => {
                const names = JSON.parse(readFileSync(directory + board.page.landmassNames, "utf8")) as {borders: string}
                expect(names.borders).toBe(board.files.borders)
            })
        })
    }
})

describe("boardOf", () => {
    it("finds a board the page carries", () => {
        expect(boardOf(FIRST_BOARD)).toBe(BOARDS[FIRST_BOARD])
        expect(boardNamed(FIRST_BOARD)).toBe(BOARDS[FIRST_BOARD])
    })

    it("refuses a board this build does not carry, by its name", () => {
        expect(boardNamed("earth-9")).toBeUndefined()
        expect(boardNamed("toString")).toBeUndefined()
        expect(() => boardOf("earth-9")).toThrow(UnknownBoardError)
        expect(() => boardOf("earth-9")).toThrow(/earth-9/)
    })
})
