import {describe, expect, it, vi} from "vitest"
import {Me} from "../../backends/account.ts"
import {FakeMarketingBackend} from "../../backends/fakeMarketingBackend.ts"
import {MarketingBackend, MarketingError, MarketingFailure} from "../../backends/marketing.ts"
import {SeasonEmailsStore} from "./seasonEmailsStore.ts"

const linked: Me = {linked: ["google"]}
const guest: Me = {linked: []}

function refusing(failure: MarketingFailure): MarketingBackend {
    const fake = new FakeMarketingBackend(["ada@example.com"])
    return {
        offered: () => fake.offered(),
        subscription: () => fake.subscription(),
        subscribe: async () => {
            throw new MarketingError(failure)
        },
        unsubscribe: async () => {
            throw new MarketingError(failure)
        },
    }
}

describe("SeasonEmailsStore", () => {
    it("stays hidden while nobody may sign in", async () => {
        const store = new SeasonEmailsStore(new FakeMarketingBackend())

        await store.follow(undefined)

        expect(store.state()).toEqual({kind: "hidden"})
    })

    it("offers a guest the button when the server has season emails", async () => {
        const store = new SeasonEmailsStore(new FakeMarketingBackend())

        await store.follow(guest)

        expect(store.state()).toEqual({kind: "guest"})
    })

    it("hides from a guest what the server does not offer", async () => {
        const backend = new FakeMarketingBackend()
        vi.spyOn(backend, "offered").mockResolvedValue(false)
        const store = new SeasonEmailsStore(backend)

        await store.follow(guest)

        expect(store.state()).toEqual({kind: "hidden"})
    })

    it("fills a signed-in player's field with the address the server suggests", async () => {
        const store = new SeasonEmailsStore(new FakeMarketingBackend(["ada@example.com"]))

        await store.follow(linked)

        expect(store.state()).toEqual({kind: "ready", state: "none", address: "ada@example.com"})
    })

    it("hides when the server has no season emails, or cannot say", async () => {
        const off = new FakeMarketingBackend()
        vi.spyOn(off, "subscription").mockResolvedValue(undefined as never)
        const offStore = new SeasonEmailsStore(off)
        await offStore.follow(linked)
        expect(offStore.state()).toEqual({kind: "hidden"})

        const failing = new FakeMarketingBackend()
        vi.spyOn(failing, "subscription").mockRejectedValue(new Error("network"))
        vi.spyOn(console, "error").mockImplementation(() => {})
        const failingStore = new SeasonEmailsStore(failing)
        await failingStore.follow(linked)
        expect(failingStore.state()).toEqual({kind: "hidden"})
    })

    it("shows nothing of the last account while it reads the new one", async () => {
        const store = new SeasonEmailsStore(new FakeMarketingBackend(["ada@example.com"]))
        await store.follow(guest)
        expect(store.state()).toEqual({kind: "guest"})

        const reading = store.follow(linked)
        expect(store.state()).toEqual({kind: "hidden"})
        await reading

        expect(store.state()).toEqual({kind: "ready", state: "none", address: "ada@example.com"})
    })

    it("reads once per account, however many controls follow it", async () => {
        const backend = new FakeMarketingBackend(["ada@example.com"])
        const read = vi.spyOn(backend, "subscription")
        const store = new SeasonEmailsStore(backend)

        await store.follow(linked)
        await store.follow(linked)

        expect(read).toHaveBeenCalledTimes(1)
        await store.follow({linked: ["google"]})
        expect(read).toHaveBeenCalledTimes(2)
    })

    it("turns a verified address on at once", async () => {
        const store = new SeasonEmailsStore(new FakeMarketingBackend(["ada@example.com"]))
        await store.follow(linked)

        await store.optIn(" Ada@Example.com ")

        expect(store.state()).toEqual({kind: "ready", state: "active", address: "ada@example.com"})
    })

    it("keeps a typed address waiting, then reads it active once confirmed", async () => {
        const backend = new FakeMarketingBackend(["ada@example.com"])
        const store = new SeasonEmailsStore(backend)
        await store.follow(linked)

        await store.optIn("ada@work.example")
        expect(store.state()).toEqual({kind: "ready", state: "waiting", address: "ada@work.example"})

        backend.confirm()
        await store.refresh()

        expect(store.state()).toEqual({kind: "ready", state: "active", address: "ada@work.example"})
    })

    it("only asks the server again while an address waits", async () => {
        const backend = new FakeMarketingBackend(["ada@example.com"])
        const store = new SeasonEmailsStore(backend)
        await store.follow(linked)
        const read = vi.spyOn(backend, "subscription")

        await store.refresh()

        expect(read).not.toHaveBeenCalled()
    })

    it("is busy while the server answers", async () => {
        const backend = new FakeMarketingBackend(["ada@example.com"])
        const store = new SeasonEmailsStore(backend)
        await store.follow(linked)

        const pending = store.optIn("ada@example.com")
        expect(store.state()).toMatchObject({kind: "ready", state: "none", busy: true})
        await store.optIn("ada@example.com")
        await pending

        expect(store.state()).toEqual({kind: "ready", state: "active", address: "ada@example.com"})
    })

    it("turns emails off and keeps the address in the field", async () => {
        const store = new SeasonEmailsStore(new FakeMarketingBackend(["ada@example.com"]))
        await store.follow(linked)
        await store.optIn("ada@work.example")

        await store.optOut()

        expect(store.state()).toEqual({kind: "ready", state: "none", address: "ada@work.example"})
    })

    it("keeps the state and says why when the server refuses", async () => {
        for (const failure of ["invalid", "unavailable", "tooManyTries", "alreadySubscribed", "failed"] as const) {
            const store = new SeasonEmailsStore(refusing(failure))
            await store.follow(linked)

            await store.optIn("ada@example.com")

            expect(store.state(), failure).toEqual({kind: "ready", state: "none", address: "ada@example.com", failure})
        }
    })

    it("falls back to the guest button when the account is no longer signed in", async () => {
        const store = new SeasonEmailsStore(refusing("guest"))
        await store.follow(linked)

        await store.optIn("ada@example.com")

        expect(store.state()).toEqual({kind: "guest"})
    })

    it("hides once the server turns season emails off", async () => {
        const store = new SeasonEmailsStore(refusing("off"))
        await store.follow(linked)

        await store.optIn("ada@example.com")

        expect(store.state()).toEqual({kind: "hidden"})
    })

    it("drops an answer for an account the menu no longer shows", async () => {
        const backend = new FakeMarketingBackend(["ada@example.com"])
        const store = new SeasonEmailsStore(backend)
        await store.follow(linked)

        const pending = store.optIn("ada@example.com")
        await store.follow(undefined)
        await pending

        expect(store.state()).toEqual({kind: "hidden"})
    })
})
