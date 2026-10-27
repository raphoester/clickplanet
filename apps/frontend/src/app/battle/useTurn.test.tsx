// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {PLAIN_TURN, Turn, TurnBackend} from "../../backends/turn.ts"
import {FakeTurnBackend} from "../../backends/fakeTurnBackend.ts"
import {TURN_REREAD_MS} from "../../domain/battle.ts"
import {useTurn} from "./useTurn.ts"

const SECOND = 1000

let latest: Turn | undefined

function Harness({backend}: {backend?: TurnBackend}) {
    latest = useTurn(backend)
    return null
}

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe("useTurn", () => {
    it("knows nothing with no backend", () => {
        render(<Harness/>)
        expect(latest).toBeUndefined()
    })

    it("holds the turn the backend answers", async () => {
        await act(async () => {
            render(<Harness backend={new FakeTurnBackend()}/>)
        })
        expect(latest).toEqual(PLAIN_TURN)
    })

    it("reads again when the backend tells of a change", async () => {
        const backend = new FakeTurnBackend()
        await act(async () => {
            render(<Harness backend={backend}/>)
        })

        await act(async () => {
            backend.startBattle("entry-final", Date.now(), Date.now() + 30 * 60_000, "Triple points.")
        })

        expect(latest?.kind).toBe("battle")
        expect(latest?.description).toBe("Triple points.")
    })

    it("reads again at the entry's end, and a minute later otherwise", async () => {
        vi.useFakeTimers({now: Date.UTC(2026, 9, 31, 21)})
        const endsAt = Date.now() + 10 * SECOND
        const turn = vi.fn(async (): Promise<Turn> => ({kind: "battle", since: Date.now(), endsAt, description: "", entryId: "entry-final"}))
        const backend: TurnBackend = {turn, changes: () => () => {}}
        await act(async () => {
            render(<Harness backend={backend}/>)
        })
        expect(turn).toHaveBeenCalledTimes(1)

        await act(async () => {
            vi.advanceTimersByTime(10 * SECOND)
        })
        expect(turn).toHaveBeenCalledTimes(2)

        await act(async () => {
            vi.advanceTimersByTime(TURN_REREAD_MS - SECOND)
        })
        expect(turn).toHaveBeenCalledTimes(2)

        await act(async () => {
            vi.advanceTimersByTime(SECOND)
        })
        expect(turn).toHaveBeenCalledTimes(3)
    })

    it("says nothing when the read fails", async () => {
        const error = vi.spyOn(console, "error").mockImplementation(() => {})
        await act(async () => {
            render(<Harness backend={{turn: () => Promise.reject(new Error("down")), changes: () => () => {}}}/>)
        })
        expect(latest).toBeUndefined()
        expect(error).toHaveBeenCalled()
    })
})
