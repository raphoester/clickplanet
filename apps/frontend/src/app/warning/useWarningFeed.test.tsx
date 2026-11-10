// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {StrictMode} from "react"
import {WarningsListener} from "../../backends/antibot.ts"
import {TOKEN_CHECK_MS} from "../useHeldIdentity.ts"
import {useWarningFeed} from "./useWarningFeed.ts"
import {useWarning} from "./warningStore.ts"

function Harness({backend}: {backend: WarningsListener}) {
    useWarningFeed(backend)
    return null
}

function listening(token: string | undefined) {
    const stream = {
        warn: (): void => {},
        stop: vi.fn(),
    }
    const backend = {
        heldIdentity: vi.fn((): string | undefined => token),
        listenForWarnings: vi.fn((onWarned: () => void) => {
            stream.warn = () => act(() => onWarned())
            return stream.stop
        }),
    }
    return {backend, stream}
}

const tick = () => act(() => vi.advanceTimersByTime(TOKEN_CHECK_MS))

afterEach(() => {
    cleanup()
    vi.useRealTimers()
    useWarning.setState(useWarning.getInitialState(), true)
})

describe("useWarningFeed", () => {
    it("puts the warning the stream sends in the store, for a guest too", () => {
        const {backend, stream} = listening("token-1")
        render(<Harness backend={backend}/>)

        stream.warn()

        expect(useWarning.getState().warned).toBe(true)
    })

    it("waits for a token, then listens", () => {
        vi.useFakeTimers()
        const {backend} = listening(undefined)
        render(<Harness backend={backend}/>)
        expect(backend.listenForWarnings).not.toHaveBeenCalled()

        backend.heldIdentity.mockReturnValue("token-1")
        tick()

        expect(backend.listenForWarnings).toHaveBeenCalledTimes(1)
    })

    it("listens again under a new token, and empties the store when a stream stops", () => {
        vi.useFakeTimers()
        const {backend, stream} = listening("token-1")
        render(<Harness backend={backend}/>)
        stream.warn()

        backend.heldIdentity.mockReturnValue("token-2")
        tick()

        expect(stream.stop).toHaveBeenCalledTimes(1)
        expect(backend.listenForWarnings).toHaveBeenCalledTimes(2)
        expect(useWarning.getState().warned).toBe(false)
    })

    it("stops and empties the store once it unmounts", () => {
        const {backend, stream} = listening("token-1")
        const view = render(<Harness backend={backend}/>)
        stream.warn()

        view.unmount()

        expect(stream.stop).toHaveBeenCalledTimes(1)
        expect(useWarning.getState().warned).toBe(false)
    })

    it("follows one stream in StrictMode", () => {
        const {backend, stream} = listening("token-1")
        render(<StrictMode><Harness backend={backend}/></StrictMode>)

        expect(backend.listenForWarnings).toHaveBeenCalledTimes(2)
        expect(stream.stop).toHaveBeenCalledTimes(1)
    })
})
