// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {renderHook} from "@testing-library/react"
import {CrazyGamesPage} from "../../backends/crazyGames.ts"
import {Countries, Country} from "../../domain/countries.ts"
import {Allegiance} from "../../domain/allegiance.ts"
import {ROOM_SETTLE_MS, useInvites, useRoom} from "./useCrazyGames.ts"

const FRANCE = Countries.get("fr")!
const GERMANY = Countries.get("de")!

function fakePage() {
    const invites = new Set<(flag: Country) => void>()
    const page: CrazyGamesPage = {
        user: {token: async () => undefined, prompt: async () => false, listenForSignIn: () => () => undefined},
        room: {
            report: vi.fn(),
            listenForInvites: (listener) => {
                invites.add(listener)
                return () => invites.delete(listener)
            },
            inviteLink: async () => undefined,
        },
        settings: () => ({chat: true, audio: true}),
        listenForSettings: () => () => undefined,
        started: vi.fn(),
    }
    return {page, invite: (flag: Country) => invites.forEach((listener) => listener(flag)), invites}
}

describe("useRoom", () => {
    beforeEach(() => {
        vi.useFakeTimers()
    })

    afterEach(() => {
        vi.useRealTimers()
    })

    it("reports the flag once the globe is on screen and the flag has settled, then each change", () => {
        const {page} = fakePage()
        const {rerender} = renderHook(({ready, flag}) => useRoom(page, ready, flag), {
            initialProps: {ready: false, flag: "fr"},
        })
        vi.advanceTimersByTime(ROOM_SETTLE_MS)
        expect(page.room.report).not.toHaveBeenCalled()

        rerender({ready: true, flag: "fr"})
        rerender({ready: true, flag: "de"})
        vi.advanceTimersByTime(ROOM_SETTLE_MS)
        expect(vi.mocked(page.room.report).mock.calls).toEqual([["de"]])

        rerender({ready: true, flag: "it"})
        vi.advanceTimersByTime(ROOM_SETTLE_MS)
        expect(vi.mocked(page.room.report).mock.calls).toEqual([["de"], ["it"]])
    })
})

describe("useInvites", () => {
    it("plays for each invite's flag while the player holds none", () => {
        const {page, invite} = fakePage()
        const play = vi.fn()
        renderHook(() => useInvites(page, undefined, play))

        invite(FRANCE)
        invite(GERMANY)

        expect(play.mock.calls).toEqual([[FRANCE], [GERMANY]])
    })

    it("keeps the flag held", () => {
        const {page, invite} = fakePage()
        const play = vi.fn()
        const {rerender} = renderHook(({held}) => useInvites(page, held, play), {
            initialProps: {held: undefined as Allegiance | undefined},
        })

        rerender({held: {countryId: "de"}})
        invite(FRANCE)

        expect(play).not.toHaveBeenCalled()
    })

    it("stops listening when the page goes", () => {
        const {page, invites} = fakePage()
        const {unmount} = renderHook(() => useInvites(page, undefined, vi.fn()))

        unmount()

        expect(invites.size).toBe(0)
    })
})
