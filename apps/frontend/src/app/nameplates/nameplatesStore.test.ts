import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import {AccountNameplate, MAX_NAMEPLATES_ASKED, Nameplate, NameplatesBackend} from "../../backends/nameplates.ts"
import {NameColor} from "../../backends/player.ts"
import {NAMEPLATE_BATCH_MS, NAMEPLATE_FRESH_MS, NAMEPLATE_RETRY_MS, NAMEPLATE_WAIT_MS, useNameplates} from "./nameplatesStore.ts"

const ANA: Nameplate = {name: "Ana", color: NameColor.PINK, admin: true, guest: false}
const KIRAN: Nameplate = {name: "kiran_07", color: NameColor.UNSPECIFIED, admin: false, guest: false}
const GUEST: Nameplate = {name: "guest_91aa3d", color: NameColor.UNSPECIFIED, admin: false, guest: true}

type Call = {ids: readonly string[], resolve: (nameplates: AccountNameplate[]) => void, reject: (e: unknown) => void}

// A backend whose every call waits for the test to answer it.
function pending(): NameplatesBackend & {calls: Call[]} {
    const calls: Call[] = []
    return {
        calls,
        nameplates: (ids) => new Promise((resolve, reject) => calls.push({ids, resolve, reject})),
    }
}

function answering(nameplates: Record<string, Nameplate>): NameplatesBackend & {asked: (readonly string[])[]} {
    const asked: (readonly string[])[] = []
    return {
        asked,
        nameplates: async (ids) => {
            asked.push(ids)
            return ids.flatMap((accountId) => nameplates[accountId] ? [{accountId, ...nameplates[accountId]}] : [])
        },
    }
}

const store = () => useNameplates.getState()

async function frame() {
    await vi.advanceTimersByTimeAsync(NAMEPLATE_BATCH_MS)
}

describe("nameplatesStore", () => {
    beforeEach(() => {
        vi.useFakeTimers()
        vi.spyOn(console, "error").mockImplementation(() => {})
    })

    afterEach(() => {
        store().serve(undefined)
        vi.useRealTimers()
        vi.restoreAllMocks()
    })

    it("reads the accounts asked in one frame in one call, and keeps each nameplate by account", async () => {
        const backend = answering({ana: ANA, guest: GUEST})
        store().serve(backend)

        store().ask("ana")
        store().ask("guest")
        store().ask("ana")
        await frame()

        expect(backend.asked).toEqual([["ana", "guest"]])
        expect(store().nameplates).toEqual({ana: ANA, guest: GUEST})
    })

    it("keeps an account left out as nobody to name, and does not ask again while fresh", async () => {
        const backend = answering({})
        store().serve(backend)

        store().ask("deleted")
        await frame()
        store().ask("deleted")
        await frame()

        expect(store().nameplates.deleted).toBeNull()
        expect(backend.asked).toEqual([["deleted"]])
    })

    it("never asks for an empty id: a message kept before accounts", async () => {
        const backend = answering({})
        store().serve(backend)

        store().ask("")
        await frame()

        expect(backend.asked).toEqual([])
    })

    it("cuts a frame of more accounts than the server takes into several calls", async () => {
        const backend = answering({})
        store().serve(backend)

        const ids = Array.from({length: MAX_NAMEPLATES_ASKED * 2 + 5}, (_, i) => `account-${i}`)
        ids.forEach((id) => store().ask(id))
        await frame()

        expect(backend.asked.map((call) => call.length)).toEqual([MAX_NAMEPLATES_ASKED, MAX_NAMEPLATES_ASKED, 5])
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
        backend.calls[0].resolve([{accountId: "ana", ...ANA}])
        await vi.advanceTimersByTimeAsync(0)
        expect(store().nameplates.ana).toEqual(ANA)
    })

    it("reads a nameplate again once it is older than the freshness, and keeps its object when it is the same", async () => {
        const backend = answering({ana: ANA})
        store().serve(backend)
        store().ask("ana")
        await frame()
        const first = store().nameplates.ana

        await vi.advanceTimersByTimeAsync(NAMEPLATE_FRESH_MS - NAMEPLATE_BATCH_MS - 1)
        store().ask("ana")
        await frame()
        expect(backend.asked).toHaveLength(1)

        await vi.advanceTimersByTimeAsync(NAMEPLATE_BATCH_MS)
        store().ask("ana")
        await frame()
        expect(backend.asked).toHaveLength(2)
        expect(store().nameplates.ana).toBe(first)
    })

    it("shows nothing new when a read fails, keeps what it had, and tries again later", async () => {
        const backend = pending()
        store().serve(backend)
        store().ask("ana")
        store().ask("kiran")
        await frame()
        backend.calls[0].resolve([{accountId: "ana", ...ANA}])
        await vi.advanceTimersByTimeAsync(NAMEPLATE_FRESH_MS)

        store().ask("ana")
        store().ask("kiran")
        await frame()
        backend.calls[1].reject(new Error("Unimplemented"))
        await vi.advanceTimersByTimeAsync(0)

        expect(store().nameplates.ana).toEqual(ANA)
        expect(store().nameplates.kiran).toBeNull()

        store().ask("kiran")
        await frame()
        expect(backend.calls).toHaveLength(2)

        await vi.advanceTimersByTimeAsync(NAMEPLATE_RETRY_MS)
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

        expect(store().nameplates.ana).toBeUndefined()
    })

    it("forgets every nameplate once nothing serves them, and drops a read answered after", async () => {
        const backend = pending()
        store().serve(backend)
        store().ask("ana")
        await frame()

        store().serve(undefined)
        backend.calls[0].resolve([{accountId: "ana", ...ANA}])
        await vi.advanceTimersByTimeAsync(0)

        expect(store().nameplates).toEqual({})
    })

    it("knows the roster's lines at once, reads none of them while fresh, and drops a read sent before", async () => {
        const backend = pending()
        store().serve(backend)
        store().ask("ana")
        await frame()

        await vi.advanceTimersByTimeAsync(1)
        store().learn([{accountId: "ana", ...ANA, name: "Ana_B"}, {accountId: "kiran", ...KIRAN}])
        backend.calls[0].resolve([{accountId: "ana", ...ANA}])
        await vi.advanceTimersByTimeAsync(0)
        store().ask("kiran")
        await frame()

        expect(store().nameplates).toEqual({ana: {...ANA, name: "Ana_B"}, kiran: KIRAN})
        expect(backend.calls).toHaveLength(1)
    })

    it("shows the page's own new name and color at once, and keeps the crown a read gave", async () => {
        const backend = answering({ana: ANA})
        store().serve(backend)
        store().ask("ana")
        await frame()

        store().own("ana", "Ana_B", NameColor.TEAL)

        expect(store().nameplates.ana).toEqual({name: "Ana_B", color: NameColor.TEAL, admin: true, guest: false})
    })

    it("names the page's own account from its profile at once, and reads whether it is an admin", async () => {
        const backend = answering({ana: ANA})
        store().serve(backend)

        store().own("ana", "Ana", NameColor.PINK)
        expect(store().nameplates.ana).toEqual({...ANA, admin: false})
        await vi.advanceTimersByTimeAsync(1)
        await frame()

        expect(backend.asked).toEqual([["ana"]])
        expect(store().nameplates.ana).toEqual(ANA)
    })

    it("names a guest that signed in as a player", async () => {
        store().serve(answering({}))
        store().learn([{accountId: "me", ...GUEST}])

        store().own("me", "Nova", NameColor.UNSPECIFIED)

        expect(store().nameplates.me).toEqual({name: "Nova", color: NameColor.UNSPECIFIED, admin: false, guest: false})
    })

    it("settles a wait once every account is read, and at once for accounts known", async () => {
        const backend = pending()
        store().serve(backend)
        store().learn([{accountId: "ana", ...ANA}])
        const known = vi.fn()
        const waited = vi.fn()

        void store().whenRead(["ana", ""]).then(known)
        void store().whenRead(["ana", "kiran"]).then(waited)
        await frame()
        expect(known).toHaveBeenCalled()
        expect(waited).not.toHaveBeenCalled()

        backend.calls[0].resolve([{accountId: "kiran", ...KIRAN}])
        await vi.advanceTimersByTimeAsync(0)
        expect(waited).toHaveBeenCalled()
    })

    it("settles a wait whose read hangs after the wait", async () => {
        store().serve(pending())
        const waited = vi.fn()

        void store().whenRead(["ana"]).then(waited)
        await vi.advanceTimersByTimeAsync(NAMEPLATE_WAIT_MS - 1)
        expect(waited).not.toHaveBeenCalled()

        await vi.advanceTimersByTimeAsync(1)
        expect(waited).toHaveBeenCalled()
    })
})
