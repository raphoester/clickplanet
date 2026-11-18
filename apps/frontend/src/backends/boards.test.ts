import {describe, expect, it, vi} from "vitest"
import {BoardChanges, BoardReload, CACHED_MS, ReloadMemory, sessionMemory, StaleBundleError} from "./boards.ts"

const CARRIED = new Set(["earth-1", "siege-1"])

function kept(start?: BoardReload): ReloadMemory & {held: BoardReload | undefined} {
    const memory = {
        held: start,
        read: () => memory.held,
        write: (reload: BoardReload) => {
            memory.held = reload
        },
        forget: () => {
            memory.held = undefined
        },
    }
    return memory
}

// A clock the waits move on.
function changesOf(memory = kept(), start = 1_000_000) {
    let now = start
    const reload = vi.fn()
    const changes = new BoardChanges({
        reload,
        carries: (board) => CARRIED.has(board),
        memory,
        now: () => now,
        wait: async (ms) => {
            now += ms
        },
    })
    return {changes, reload, memory, at: () => now, pass: (ms: number) => {
        now += ms
    }}
}

// GetMap's answers in turn, the last one again and again.
function answers(...boards: string[]) {
    let read = 0
    const getMap = vi.fn(async () => boards[Math.min(read++, boards.length - 1)])
    return getMap
}

describe("BoardChanges.choose", () => {
    it("draws the board GetMap names", async () => {
        const {changes, reload} = changesOf()

        await expect(changes.choose(answers("earth-1"))).resolves.toBe("earth-1")
        expect(changes.board).toBe("earth-1")
        expect(reload).not.toHaveBeenCalled()
    })

    it("reads an answer from a server before boards were named as earth-1", async () => {
        const {changes} = changesOf()

        await expect(changes.choose(answers(""))).resolves.toBe("earth-1")
    })

    it("after a reload for a board, reads GetMap again while it names the board before: the browser's copy", async () => {
        const memory = kept({board: "siege-1", at: 1_000_000 - 2_000})
        const {changes, reload} = changesOf(memory)
        const getMap = answers("earth-1", "earth-1", "siege-1")

        await expect(changes.choose(getMap)).resolves.toBe("siege-1")
        expect(getMap).toHaveBeenCalledTimes(3)
        expect(reload).not.toHaveBeenCalled()
        expect(memory.held).toBeUndefined()
    })

    it("takes the board GetMap keeps naming once the browser's copy is past, and does not reload again", async () => {
        const memory = kept({board: "siege-1", at: 1_000_000})
        const {changes, reload, at} = changesOf(memory)
        const getMap = answers("earth-1")

        await expect(changes.choose(getMap)).resolves.toBe("earth-1")
        expect(at() - 1_000_000).toBeGreaterThan(CACHED_MS)
        expect(getMap.mock.calls.length).toBeLessThan(10)
        expect(reload).not.toHaveBeenCalled()
    })

    it("forgets a reload long past: the page is a load of its own", async () => {
        const memory = kept({board: "siege-1", at: 1_000_000 - 120_000})
        const {changes} = changesOf(memory)
        const getMap = answers("earth-1")

        await expect(changes.choose(getMap)).resolves.toBe("earth-1")
        expect(getMap).toHaveBeenCalledTimes(1)
    })

    it("trusts the stream's opening over a read the browser may have kept", async () => {
        const memory = kept({board: "siege-1", at: 1_000_000})
        const {changes} = changesOf(memory)
        changes.opened("siege-1")
        const getMap = answers("earth-1")

        await expect(changes.choose(getMap)).resolves.toBe("siege-1")
        expect(getMap).toHaveBeenCalledTimes(1)
    })

    it("reloads once for a board this build does not carry, then fails", async () => {
        const memory = kept()
        const first = changesOf(memory)
        const choosing = first.changes.choose(answers("earth-9"))

        await Promise.resolve()
        await Promise.resolve()
        expect(first.reload).toHaveBeenCalledTimes(1)
        expect(memory.held).toEqual({board: "earth-9", at: 1_000_000})
        const settled = vi.fn()
        void choosing.then(settled, settled)
        await new Promise((resolve) => setTimeout(resolve, 0))
        expect(settled).not.toHaveBeenCalled()

        const second = changesOf(memory, 1_003_000)
        await expect(second.changes.choose(answers("earth-9"))).rejects.toBeInstanceOf(StaleBundleError)
        expect(second.reload).not.toHaveBeenCalled()
    })
})

describe("BoardChanges.opened", () => {
    it("reloads once when the stream opens on another board than the page drew", async () => {
        const {changes, reload, memory} = changesOf()
        await changes.choose(answers("earth-1"))

        expect(changes.opened("earth-1")).toBe(false)
        expect(changes.opened("")).toBe(false)
        expect(reload).not.toHaveBeenCalled()

        expect(changes.opened("siege-1")).toBe(true)
        expect(changes.opened("earth-1")).toBe(true)
        expect(changes.opened("siege-1")).toBe(true)
        expect(reload).toHaveBeenCalledTimes(1)
        expect(memory.held).toEqual({board: "siege-1", at: 1_000_000})
    })
})

describe("BoardChanges.answered", () => {
    it("draws an answer on the board the page drew", async () => {
        const {changes} = changesOf()
        await changes.choose(answers("earth-1"))

        expect(changes.answered("earth-1")).toBe("drawn")
    })

    it("reads another board again past the browser's copy, and reloads once that read still names it", async () => {
        const {changes, reload, pass} = changesOf()
        await changes.choose(answers("earth-1"))

        expect(changes.answered("siege-1")).toBe("readAgain")
        pass(CACHED_MS - 1)
        expect(changes.answered("siege-1")).toBe("readAgain")
        expect(reload).not.toHaveBeenCalled()

        pass(1)
        expect(changes.answered("siege-1")).toBe("reloading")
        expect(changes.answered("earth-1")).toBe("reloading")
        expect(reload).toHaveBeenCalledTimes(1)
    })

    it("takes the drawn board back for the browser's copy it was", async () => {
        const {changes, reload, pass} = changesOf()
        await changes.choose(answers("earth-1"))

        expect(changes.answered("siege-1")).toBe("readAgain")
        pass(CACHED_MS)
        expect(changes.answered("earth-1")).toBe("drawn")
        expect(changes.answered("siege-1")).toBe("readAgain")
        expect(reload).not.toHaveBeenCalled()
    })
})

describe("sessionMemory", () => {
    it("keeps a reload for the next load of the tab", () => {
        const items = new Map<string, string>()
        const storage = {
            getItem: (key: string) => items.get(key) ?? null,
            setItem: (key: string, value: string) => void items.set(key, value),
            removeItem: (key: string) => void items.delete(key),
        } as unknown as Storage
        const memory = sessionMemory(() => storage)

        memory.write({board: "siege-1", at: 5})
        expect(sessionMemory(() => storage).read()).toEqual({board: "siege-1", at: 5})
        memory.forget()
        expect(memory.read()).toBeUndefined()
    })

    it("keeps nothing, and throws nothing, with no storage", () => {
        const memory = sessionMemory(() => {
            throw new Error("blocked")
        })

        memory.write({board: "siege-1", at: 5})
        expect(memory.read()).toBeUndefined()
        expect(() => memory.forget()).not.toThrow()
    })
})
