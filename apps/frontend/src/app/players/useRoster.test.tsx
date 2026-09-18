// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {PresenceBackend, RosterEntry, RosterEvent} from "../../backends/player.ts"
import {RosterState, useRoster} from "./useRoster.ts"

const ana: RosterEntry = {key: "k1", name: "ana", countryCode: "fr", guest: false, admin: false}
const bo: RosterEntry = {key: "k2", name: "guest_Bo", countryCode: "de", guest: true, admin: false}

let latest: RosterState

function Harness({backend}: {backend?: PresenceBackend}) {
    latest = useRoster(backend)
    return null
}

/** A backend whose stream the test drives by hand. */
function streaming() {
    const stream = {
        emit: (event: RosterEvent): void => void event,
        unavailable: () => {},
        stop: vi.fn(),
    }
    const backend = {
        heldSession: vi.fn(),
        announce: vi.fn(),
        leave: vi.fn(),
        listenForRoster: vi.fn((onEvent: (event: RosterEvent) => void, onUnavailable: () => void) => {
            stream.emit = (event) => act(() => onEvent(event))
            stream.unavailable = () => act(() => onUnavailable())
            return stream.stop
        }),
    }
    return {backend, stream}
}

afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
})

describe("useRoster", () => {
    it("is unavailable with no backend", () => {
        render(<Harness/>)
        expect(latest).toEqual({kind: "unavailable"})
    })

    it("is loading until the whole roster arrives, then follows each change", () => {
        const {backend, stream} = streaming()
        render(<Harness backend={backend}/>)
        expect(latest).toEqual({kind: "loading"})

        stream.emit({kind: "entry", entry: bo})
        expect(latest).toEqual({kind: "loading"})

        stream.emit({kind: "roster", entries: [ana]})
        expect(latest).toEqual({kind: "ready", entries: [ana]})

        stream.emit({kind: "entry", entry: bo})
        expect(latest).toEqual({kind: "ready", entries: [ana, bo]})

        stream.emit({kind: "entry", entry: {...bo, name: "Zed", guest: false}})
        expect(latest).toEqual({kind: "ready", entries: [ana, {...bo, name: "Zed", guest: false}]})

        stream.emit({kind: "left", key: "k1"})
        expect(latest).toEqual({kind: "ready", entries: [{...bo, name: "Zed", guest: false}]})
    })

    it("starts over from the roster a reconnect sends", () => {
        const {backend, stream} = streaming()
        render(<Harness backend={backend}/>)
        stream.emit({kind: "roster", entries: [ana, bo]})

        stream.emit({kind: "roster", entries: [bo]})

        expect(latest).toEqual({kind: "ready", entries: [bo]})
    })

    it("hides the list for good on a server without the live roster", () => {
        const {backend, stream} = streaming()
        render(<Harness backend={backend}/>)
        stream.emit({kind: "roster", entries: [ana]})

        stream.unavailable()

        expect(latest).toEqual({kind: "unavailable"})
    })

    it("stops following the stream when it unmounts", () => {
        const {backend, stream} = streaming()
        const view = render(<Harness backend={backend}/>)

        view.unmount()

        expect(stream.stop).toHaveBeenCalledTimes(1)
    })
})
