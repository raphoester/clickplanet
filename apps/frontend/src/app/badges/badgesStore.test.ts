import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {AccountBadge, BadgesBackend, MAX_BADGES_ASKED} from "../../backends/badges.ts"
import {PlayerTitle} from "../../backends/title.ts"
import {BADGE_BATCH_MS, BADGE_FRESH_MS, BADGE_RETRY_MS, NO_BADGE, useBadges} from "./badgesStore.ts"

const OG: PlayerTitle = {id: "og", name: "OG"}
const WARLORD: PlayerTitle = {id: "warlord", name: "Warlord", rank: {trackId: "conquest", trackName: "Conquest", number: 3, count: 5}}

type Call = {ids: readonly string[], resolve: (badges: AccountBadge[]) => void, reject: (e: unknown) => void}

// A backend whose every call waits for the test to answer it.
function pending(): BadgesBackend & {calls: Call[]} {
    const calls: Call[] = []
    return {
        calls,
        badges: (ids) => new Promise((resolve, reject) => calls.push({ids, resolve, reject})),
    }
}

function answering(badges: Record<string, Omit<AccountBadge, "accountId">>): BadgesBackend & {asked: (readonly string[])[]} {
    const asked: (readonly string[])[] = []
    return {
        asked,
        badges: async (ids) => {
            asked.push(ids)
            return ids.flatMap((accountId) => badges[accountId] ? [{accountId, ...badges[accountId]}] : [])
        },
    }
}

const store = () => useBadges.getState()

async function frame() {
    await vi.advanceTimersByTimeAsync(BADGE_BATCH_MS)
}

describe("badgesStore", () => {
    beforeEach(() => {
        vi.useFakeTimers()
        vi.spyOn(console, "error").mockImplementation(() => {})
    })

    afterEach(() => {
        store().serve(undefined)
        vi.useRealTimers()
        vi.restoreAllMocks()
    })

    it("reads the accounts asked in one frame in one call, and keeps each badge by account", async () => {
        const backend = answering({ana: {streak: 12, wornTitle: WARLORD}, kiran: {streak: 3}})
        store().serve(backend)

        store().ask("ana")
        store().ask("kiran")
        store().ask("ana")
        await frame()

        expect(backend.asked).toEqual([["ana", "kiran"]])
        expect(store().badges).toEqual({ana: {streak: 12, wornTitle: WARLORD}, kiran: {streak: 3}})
    })

    it("keeps an account left out as one with nothing to show, and does not ask again while fresh", async () => {
        const backend = answering({})
        store().serve(backend)

        store().ask("guest")
        await frame()
        store().ask("guest")
        await frame()

        expect(store().badges.guest).toBe(NO_BADGE)
        expect(backend.asked).toEqual([["guest"]])
    })

    it("never asks for an empty id: a message kept before accounts, or of a deleted one", async () => {
        const backend = answering({})
        store().serve(backend)

        store().ask("")
        await frame()

        expect(backend.asked).toEqual([])
    })

    it("cuts a frame of more accounts than the server takes into several calls", async () => {
        const backend = answering({})
        store().serve(backend)

        const ids = Array.from({length: MAX_BADGES_ASKED * 2 + 5}, (_, i) => `account-${i}`)
        ids.forEach((id) => store().ask(id))
        await frame()

        expect(backend.asked.map((call) => call.length)).toEqual([MAX_BADGES_ASKED, MAX_BADGES_ASKED, 5])
        expect(backend.asked.flat()).toEqual(ids)
    })

    it("does not ask again for an account being read", async () => {
        const backend = pending()
        store().serve(backend)

        store().ask("ana")
        await frame()
        store().ask("ana")
        await frame()

        expect(backend.calls).toHaveLength(1)
        backend.calls[0].resolve([{accountId: "ana", streak: 2}])
        await vi.advanceTimersByTimeAsync(0)
        expect(store().badges.ana).toEqual({streak: 2})
    })

    it("reads a badge again once it is older than the freshness, the next time it is asked", async () => {
        const backend = answering({ana: {streak: 2}})
        store().serve(backend)
        store().ask("ana")
        await frame()

        await vi.advanceTimersByTimeAsync(BADGE_FRESH_MS - BADGE_BATCH_MS - 1)
        store().ask("ana")
        await frame()
        expect(backend.asked).toHaveLength(1)

        await vi.advanceTimersByTimeAsync(BADGE_BATCH_MS)
        store().ask("ana")
        await frame()
        expect(backend.asked).toHaveLength(2)
    })

    it("keeps the object of a badge read again the same, so nothing that shows it renders", async () => {
        const backend = answering({ana: {streak: 2, wornTitle: OG}})
        store().serve(backend)
        store().ask("ana")
        await frame()
        const first = store().badges.ana

        await vi.advanceTimersByTimeAsync(BADGE_FRESH_MS)
        store().ask("ana")
        await frame()

        expect(backend.asked).toHaveLength(2)
        expect(store().badges.ana).toBe(first)
    })

    it("shows no badge when a read fails, keeps the one it had, and tries again later", async () => {
        const backend = pending()
        store().serve(backend)
        store().ask("ana")
        store().ask("kiran")
        await frame()
        backend.calls[0].resolve([{accountId: "ana", streak: 2}])
        await vi.advanceTimersByTimeAsync(BADGE_FRESH_MS)

        store().ask("ana")
        store().ask("kiran")
        await frame()
        backend.calls[1].reject(new Error("Unimplemented"))
        await vi.advanceTimersByTimeAsync(0)

        expect(store().badges.ana).toEqual({streak: 2})
        expect(store().badges.kiran).toBe(NO_BADGE)

        store().ask("kiran")
        await frame()
        expect(backend.calls).toHaveLength(2)

        await vi.advanceTimersByTimeAsync(BADGE_RETRY_MS)
        store().ask("kiran")
        await frame()
        expect(backend.calls).toHaveLength(3)
    })

    it("shows nothing for an account whose first read fails", async () => {
        const backend = pending()
        store().serve(backend)
        store().ask("ana")
        await frame()

        backend.calls[0].reject(new Error("Unimplemented"))
        await vi.advanceTimersByTimeAsync(0)

        expect(store().badges.ana).toBeUndefined()
    })

    it("reads what was asked before a backend served it once one does", async () => {
        store().ask("ana")
        await frame()
        const backend = answering({ana: {streak: 5}})

        store().serve(backend)
        await frame()

        expect(backend.asked).toEqual([["ana"]])
        expect(store().badges.ana).toEqual({streak: 5})
    })

    it("forgets every badge once nothing serves them, and drops a read answered after", async () => {
        const backend = pending()
        store().serve(backend)
        store().ask("ana")
        await frame()

        store().serve(undefined)
        backend.calls[0].resolve([{accountId: "ana", streak: 2}])
        await vi.advanceTimersByTimeAsync(0)

        expect(store().badges).toEqual({})
    })

    it("writes the title the page wears at once, and a read sent before cannot undo it", async () => {
        const backend = pending()
        store().serve(backend)
        store().ask("me")
        await frame()

        store().wear("me", WARLORD)
        backend.calls[0].resolve([{accountId: "me", streak: 4, wornTitle: OG}])
        await vi.advanceTimersByTimeAsync(0)

        expect(store().badges.me).toEqual({streak: 0, wornTitle: WARLORD})
    })

    it("reads the rest of a badge the page only wrote part of", async () => {
        const backend = answering({me: {streak: 4, wornTitle: OG}})
        store().serve(backend)

        store().wear("me", OG)
        expect(store().badges.me).toEqual({streak: 0, wornTitle: OG})
        await vi.advanceTimersByTimeAsync(1)
        store().ask("me")
        await frame()

        expect(store().badges.me).toEqual({streak: 4, wornTitle: OG})
    })

    it("keeps the title worn when the page writes the streak it read", async () => {
        const backend = answering({me: {streak: 4, wornTitle: OG}})
        store().serve(backend)
        store().ask("me")
        await frame()

        store().setStreak("me", 5)

        expect(store().badges.me).toEqual({streak: 5, wornTitle: OG})
    })

    it("reads an account again at once when the page learns it changed", async () => {
        const backend = answering({me: {streak: 4, wornTitle: OG}})
        store().serve(backend)
        store().ask("me")
        await frame()

        store().expire("me")
        await frame()

        expect(backend.asked).toEqual([["me"], ["me"]])
    })

    it("drops a read sent before the page learned the account changed", async () => {
        const backend = pending()
        store().serve(backend)
        store().ask("me")
        await frame()

        await vi.advanceTimersByTimeAsync(1)
        store().expire("me")
        backend.calls[0].resolve([{accountId: "me", streak: 4, wornTitle: OG}])
        await vi.advanceTimersByTimeAsync(0)
        expect(store().badges.me).toBeUndefined()

        store().ask("me")
        await frame()
        expect(backend.calls).toHaveLength(2)
    })
})
