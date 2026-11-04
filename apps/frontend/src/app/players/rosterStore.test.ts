import {afterEach, describe, expect, it, vi} from "vitest"
import {NameColor, RosterEntry} from "../../backends/player.ts"
import {onlineCount, rosterEntries, useRoster} from "./rosterStore.ts"

const ana: RosterEntry = {key: "k1", accountId: "account-ana", name: "ana", countryCode: "fr", guest: false, admin: false, color: NameColor.UNSPECIFIED}
const bo: RosterEntry = {key: "k2", accountId: "account-bo", name: "guest_Bo", countryCode: "de", guest: true, admin: false, color: NameColor.UNSPECIFIED}

afterEach(() => useRoster.setState({state: {kind: "unavailable"}}))

describe("the roster", () => {
    it("lists nobody until the whole roster comes, then counts it", () => {
        const roster = useRoster.getState()
        expect(onlineCount(useRoster.getState())).toBeUndefined()

        roster.follow()
        roster.hear({kind: "entry", entry: bo})
        expect(useRoster.getState().state).toEqual({kind: "loading"})
        expect(rosterEntries(useRoster.getState())).toEqual([])

        roster.hear({kind: "roster", entries: [ana, bo]})
        expect(onlineCount(useRoster.getState())).toBe(2)
        expect(rosterEntries(useRoster.getState())).toEqual([ana, bo])
    })

    it("starts over from the roster a reconnect sends, and keeps its list while it waits for it", () => {
        const roster = useRoster.getState()
        roster.follow()
        roster.hear({kind: "roster", entries: [ana, bo]})

        roster.follow()
        expect(rosterEntries(useRoster.getState())).toEqual([ana, bo])

        roster.hear({kind: "roster", entries: [bo]})
        expect(rosterEntries(useRoster.getState())).toEqual([bo])
    })

    it("tells nobody of an event that changes nothing", () => {
        const roster = useRoster.getState()
        roster.follow()
        roster.hear({kind: "roster", entries: [ana]})
        const told = vi.fn()
        const stop = useRoster.subscribe(told)

        roster.hear({kind: "left", key: "k2"})
        roster.follow()

        expect(told).not.toHaveBeenCalled()
        stop()
    })

    it("lists nobody once lost", () => {
        const roster = useRoster.getState()
        roster.follow()
        roster.hear({kind: "roster", entries: [ana]})

        roster.lose()

        expect(useRoster.getState().state).toEqual({kind: "unavailable"})
        expect(onlineCount(useRoster.getState())).toBeUndefined()
    })
})
