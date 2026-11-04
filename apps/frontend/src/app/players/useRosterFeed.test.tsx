// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {StrictMode} from "react"
import {NameColor, PlayerTitle, PresenceBackend, RosterEntry, RosterEvent} from "../../backends/player.ts"
import {SETTLE_MS} from "../../domain/presence.ts"
import {useRoster} from "./rosterStore.ts"
import {useRosterFeed} from "./useRosterFeed.ts"

const ana: RosterEntry = {key: "k1", accountId: "account-ana", name: "ana", countryCode: "fr", guest: false, admin: false, color: NameColor.UNSPECIFIED}
const bo: RosterEntry = {key: "k2", accountId: "account-bo", name: "guest_Bo", countryCode: "de", guest: true, admin: false, color: NameColor.UNSPECIFIED}

const roster = () => useRoster.getState().state

function Harness({backend, onTitleEarned}: {backend?: PresenceBackend, onTitleEarned?: (title: PlayerTitle) => void}) {
    useRosterFeed(backend, onTitleEarned)
    return null
}

function streaming() {
    const stream = {
        emit: (event: RosterEvent): void => void event,
        unavailable: () => {},
        earn: (title: PlayerTitle): void => void title,
        stop: vi.fn(),
    }
    const backend = {
        heldSession: vi.fn((): string | undefined => undefined),
        heldIdentity: vi.fn((): string | undefined => undefined),
        announce: vi.fn(),
        leave: vi.fn(),
        listenForRoster: vi.fn((onEvent: (event: RosterEvent) => void, onUnavailable: () => void, onTitleEarned: (title: PlayerTitle) => void) => {
            stream.emit = (event) => act(() => onEvent(event))
            stream.unavailable = () => act(() => onUnavailable())
            stream.earn = (title) => act(() => onTitleEarned(title))
            return stream.stop
        }),
    }
    return {backend, stream}
}

afterEach(() => {
    cleanup()
    useRoster.setState({state: {kind: "unavailable"}})
    vi.restoreAllMocks()
    vi.useRealTimers()
})

describe("useRosterFeed", () => {
    it("lists nobody with no backend", () => {
        render(<Harness/>)
        expect(roster()).toEqual({kind: "unavailable"})
    })

    it("is loading until the whole roster arrives, then follows each change", () => {
        const {backend, stream} = streaming()
        render(<Harness backend={backend}/>)
        expect(roster()).toEqual({kind: "loading"})

        stream.emit({kind: "entry", entry: bo})
        expect(roster()).toEqual({kind: "loading"})

        stream.emit({kind: "roster", entries: [ana]})
        expect(roster()).toEqual({kind: "ready", entries: [ana]})

        stream.emit({kind: "entry", entry: bo})
        expect(roster()).toEqual({kind: "ready", entries: [ana, bo]})

        stream.emit({kind: "entry", entry: {...bo, name: "Zed", guest: false}})
        expect(roster()).toEqual({kind: "ready", entries: [ana, {...bo, name: "Zed", guest: false}]})

        stream.emit({kind: "left", key: "k1"})
        expect(roster()).toEqual({kind: "ready", entries: [{...bo, name: "Zed", guest: false}]})
    })

    it("hides the list for good on a server without the live roster", () => {
        const {backend, stream} = streaming()
        render(<Harness backend={backend}/>)
        stream.emit({kind: "roster", entries: [ana]})

        stream.unavailable()

        expect(roster()).toEqual({kind: "unavailable"})
    })

    it("stops following the stream when it unmounts, and lists nobody", () => {
        const {backend, stream} = streaming()
        const view = render(<Harness backend={backend}/>)
        stream.emit({kind: "roster", entries: [ana]})

        view.unmount()

        expect(stream.stop).toHaveBeenCalledTimes(1)
        expect(roster()).toEqual({kind: "unavailable"})
    })

    it("follows one stream in StrictMode", () => {
        const {backend, stream} = streaming()
        render(<StrictMode><Harness backend={backend}/></StrictMode>)

        expect(backend.listenForRoster).toHaveBeenCalledTimes(2)
        expect(stream.stop).toHaveBeenCalledTimes(1)
        stream.emit({kind: "roster", entries: [ana]})
        expect(roster()).toEqual({kind: "ready", entries: [ana]})
    })

    it("opens the stream again under a new token, and keeps the list it has meanwhile", () => {
        vi.useFakeTimers()
        const {backend, stream} = streaming()
        render(<Harness backend={backend}/>)
        stream.emit({kind: "roster", entries: [ana]})

        backend.heldIdentity.mockReturnValue("token-1")
        act(() => vi.advanceTimersByTime(SETTLE_MS))

        expect(stream.stop).toHaveBeenCalledTimes(1)
        expect(backend.listenForRoster).toHaveBeenCalledTimes(2)
        expect(roster()).toEqual({kind: "ready", entries: [ana]})

        act(() => vi.advanceTimersByTime(SETTLE_MS))
        expect(backend.listenForRoster).toHaveBeenCalledTimes(2)
    })

    it("hands a title earned to the callback it was last given", () => {
        const {backend, stream} = streaming()
        const first = vi.fn()
        const second = vi.fn()
        const view = render(<Harness backend={backend} onTitleEarned={first}/>)
        view.rerender(<Harness backend={backend} onTitleEarned={second}/>)

        stream.earn({id: "settler", name: "Settler"})

        expect(first).not.toHaveBeenCalled()
        expect(second).toHaveBeenCalledWith({id: "settler", name: "Settler"})
        expect(backend.listenForRoster).toHaveBeenCalledTimes(1)
    })
})
