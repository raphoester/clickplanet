// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {StrictMode} from "react"
import {PlayerLevel, PlayerTitle, ProgressionEvents, ProgressionListener} from "../../backends/progression.ts"
import {useProgressionEvents} from "./useProgressionEvents.ts"
import {TOKEN_CHECK_MS} from "../useHeldIdentity.ts"

const settler: PlayerTitle = {id: "settler", name: "Settler"}

const nine: PlayerLevel = {level: 9, xp: 7200, levelXp: 7168, nextLevelXp: 10206}

type HarnessProps = {
    backend: ProgressionListener
    signedIn: boolean
    onEarned?: (title: PlayerTitle) => void
    onLevel?: (level: PlayerLevel) => void
}

function Harness({backend, signedIn, onEarned = () => {}, onLevel = () => {}}: HarnessProps) {
    useProgressionEvents(backend, signedIn, {titleEarned: onEarned, levelReached: onLevel})
    return null
}

function listening(token: string | undefined) {
    const stream = {
        earn: (title: PlayerTitle): void => void title,
        reach: (level: PlayerLevel): void => void level,
        stop: vi.fn(),
    }
    const backend = {
        heldIdentity: vi.fn((): string | undefined => token),
        listenForProgression: vi.fn((on: ProgressionEvents) => {
            stream.earn = (title) => act(() => on.titleEarned(title))
            stream.reach = (level) => act(() => on.levelReached(level))
            return stream.stop
        }),
    }
    return {backend, stream}
}

const tick = () => act(() => vi.advanceTimersByTime(TOKEN_CHECK_MS))

afterEach(() => {
    cleanup()
    vi.useRealTimers()
})

describe("useProgressionEvents", () => {
    it("listens for the titles of a signed-in account under the token it holds", () => {
        const {backend, stream} = listening("token-1")
        const onEarned = vi.fn()
        render(<Harness backend={backend} signedIn onEarned={onEarned}/>)

        stream.earn(settler)

        expect(backend.listenForProgression).toHaveBeenCalledTimes(1)
        expect(onEarned).toHaveBeenCalledWith(settler)
    })

    it("hands each level reached to its callback", () => {
        const {backend, stream} = listening("token-1")
        const onLevel = vi.fn()
        const onEarned = vi.fn()
        render(<Harness backend={backend} signedIn onEarned={onEarned} onLevel={onLevel}/>)

        stream.reach(nine)

        expect(onLevel).toHaveBeenCalledWith(nine)
        expect(onEarned).not.toHaveBeenCalled()
    })

    it("listens for nothing for a guest, which earns no title and no level", () => {
        const {backend} = listening("token-1")
        render(<Harness backend={backend} signedIn={false}/>)

        expect(backend.listenForProgression).not.toHaveBeenCalled()
    })

    it("waits for a token, then listens", () => {
        vi.useFakeTimers()
        const {backend} = listening(undefined)
        render(<Harness backend={backend} signedIn/>)
        expect(backend.listenForProgression).not.toHaveBeenCalled()

        backend.heldIdentity.mockReturnValue("token-1")
        tick()

        expect(backend.listenForProgression).toHaveBeenCalledTimes(1)
    })

    it("listens again under a new token, and stops once none is held", () => {
        vi.useFakeTimers()
        const {backend, stream} = listening("token-1")
        render(<Harness backend={backend} signedIn/>)

        tick()
        expect(backend.listenForProgression).toHaveBeenCalledTimes(1)

        backend.heldIdentity.mockReturnValue("token-2")
        tick()
        expect(stream.stop).toHaveBeenCalledTimes(1)
        expect(backend.listenForProgression).toHaveBeenCalledTimes(2)

        backend.heldIdentity.mockReturnValue(undefined)
        tick()
        expect(stream.stop).toHaveBeenCalledTimes(2)
        expect(backend.listenForProgression).toHaveBeenCalledTimes(2)
    })

    it("stops when the account signs out", () => {
        const {backend, stream} = listening("token-1")
        const view = render(<Harness backend={backend} signedIn/>)

        view.rerender(<Harness backend={backend} signedIn={false}/>)

        expect(stream.stop).toHaveBeenCalledTimes(1)
    })

    it("hands a title to the callback it was last given, without listening again", () => {
        const {backend, stream} = listening("token-1")
        const first = vi.fn()
        const second = vi.fn()
        const view = render(<Harness backend={backend} signedIn onEarned={first}/>)
        view.rerender(<Harness backend={backend} signedIn onEarned={second}/>)

        stream.earn(settler)

        expect(first).not.toHaveBeenCalled()
        expect(second).toHaveBeenCalledWith(settler)
        expect(backend.listenForProgression).toHaveBeenCalledTimes(1)
    })

    it("follows one stream in StrictMode", () => {
        const {backend, stream} = listening("token-1")
        render(<StrictMode><Harness backend={backend} signedIn/></StrictMode>)

        expect(backend.listenForProgression).toHaveBeenCalledTimes(2)
        expect(stream.stop).toHaveBeenCalledTimes(1)
    })
})
