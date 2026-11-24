import {afterEach, beforeEach, describe, expect, it, vi} from "vitest"
import type {FlagPicture} from "../domain/flags.ts"
import type {CheckoutTarget} from "../domain/shop.ts"
import {FakeFlagsBackend} from "./fakeFlagsBackend.ts"
import {FakeFlagSubmissions} from "./fakeFlagSubmissions.ts"
import {FakeShop} from "./fakeShop.ts"
import {ShopRefusedError} from "./shop.ts"

const PICTURE: FlagPicture = {url: "blob:upload", width: 600, height: 400, stretch: false, focusX: 0.5, focusY: 0.5}
const PIN = {latitude: 48.1, longitude: 11.6}
const ORIGIN = "http://localhost:5173"
const LANDS_MS = 2_000
const DAY_MS = 24 * 60 * 60 * 1000

async function setup(linked = true) {
    const signedIn = {linked}
    const flags = new FakeFlagsBackend(async () => ({...PICTURE, url: "blob:painted"}))
    const submissions = new FakeFlagSubmissions(flags, async () => signedIn.linked, async () => PICTURE, 0)
    const shop = new FakeShop(flags, submissions, async () => signedIn.linked, ORIGIN, LANDS_MS, 0)
    return {flags, submissions, shop, signedIn}
}

// A new flag sent and approved: what the shop sells first.
async function approved(submissions: FakeFlagSubmissions): Promise<string> {
    const sent = await submissions.submitFlag({name: "Bavaria", uploadId: await submissions.uploadPicture(new Blob(["png"], {type: "image/png"})), pin: PIN})
    await submissions.approve()
    return sent.id
}

const refusalOf = (sent: Promise<unknown>) =>
    sent.then(() => "sent", (e: unknown) => e instanceof ShopRefusedError ? e.refusal : e)

beforeEach(() => {
    vi.useFakeTimers({shouldAdvanceTime: true})
})

afterEach(() => {
    vi.useRealTimers()
})

describe("FakeShop", () => {
    it("sells the week, the month and the year in euros, and none while off", async () => {
        const {shop} = await setup()
        expect(await shop.plans()).toEqual([
            {plan: "week", amount: 400, currency: "eur", renews: false},
            {plan: "month", amount: 1000, currency: "eur", renews: true},
            {plan: "year", amount: 6000, currency: "eur", renews: true},
        ])

        shop.turn(false)
        expect(await shop.plans()).toEqual([])
    })

    it("refuses as the server: a guest, no consent, the shop off, a submission not approved, a flag not founded", async () => {
        const {shop, submissions, signedIn} = await setup()
        const id = await approved(submissions)
        const target: CheckoutTarget = {kind: "submission", id}

        expect(await refusalOf(shop.checkout({plan: "week", target, startsNow: false}))).toBe("invalid")
        expect(await refusalOf(shop.checkout({plan: "week", target: {kind: "submission", id: "s-none"}, startsNow: true}))).toBe("not-found")
        expect(await refusalOf(shop.checkout({plan: "week", target: {kind: "flag", flag: "xb"}, startsNow: true}))).toBe("not-yours")
        shop.turn(false)
        expect(await refusalOf(shop.checkout({plan: "week", target, startsNow: true}))).toBe("unavailable")
        shop.turn(true)
        signedIn.linked = false
        expect(await refusalOf(shop.checkout({plan: "week", target, startsNow: true}))).toBe("not-yours")
    })

    it("comes back paid, then opens the flag a moment later, and a month renews", async () => {
        const {shop, submissions, flags} = await setup()
        const id = await approved(submissions)

        expect(await shop.checkout({plan: "month", target: {kind: "submission", id}, startsNow: true})).toBe(`${ORIGIN}/play?shop=paid`)
        expect((await submissions.mySubmissions())[0].state).toBe("approved")
        expect(await refusalOf(shop.checkout({plan: "week", target: {kind: "submission", id}, startsNow: true}))).toBe("unavailable")

        await vi.advanceTimersByTimeAsync(LANDS_MS)

        const [opened] = await submissions.mySubmissions()
        expect(opened.state).toBe("opened")
        const sold = (await flags.listFlags()).find(({code}) => code === opened.flag)?.sold
        expect(sold?.endsAt).toBeGreaterThanOrEqual(Date.now() + 29 * DAY_MS)
        expect(await shop.myRenewals()).toMatchObject([{plan: "month", target: {kind: "submission", id}, state: "renewing"}])
    })

    it("adds a week to a founded flag, and refuses more while its renewal runs", async () => {
        const {shop, submissions, flags} = await setup()
        const id = await approved(submissions)
        await submissions.openApproved(id, 7)
        const [{flag}] = await submissions.mySubmissions()
        const endsAt = () => flags.listFlags().then((listed) => listed.find(({code}) => code === flag)?.sold?.endsAt ?? 0)
        const before = await endsAt()

        await shop.checkout({plan: "week", target: {kind: "flag", flag}, startsNow: true})
        await vi.advanceTimersByTimeAsync(LANDS_MS)
        expect(await endsAt()).toBe(before + 7 * DAY_MS)
        expect(await shop.myRenewals()).toEqual([])

        await shop.checkout({plan: "year", target: {kind: "flag", flag}, startsNow: true})
        await vi.advanceTimersByTimeAsync(LANDS_MS)
        expect(await refusalOf(shop.checkout({plan: "week", target: {kind: "flag", flag}, startsNow: true}))).toBe("unavailable")
    })

    it("stops a renewal, and its time stays", async () => {
        const {shop, submissions} = await setup()
        const id = await approved(submissions)
        await shop.checkout({plan: "year", target: {kind: "submission", id}, startsNow: true})
        const [renewal] = await shop.myRenewals()

        expect(await shop.stopRenewal(renewal.id)).toEqual({...renewal, state: "stopping"})
        expect(await refusalOf(shop.stopRenewal("r-none"))).toBe("not-found")
    })

    it("refuses the next checkouts with the refusal asked on the console", async () => {
        const {shop, submissions} = await setup()
        const id = await approved(submissions)

        shop.refuse("not-found")
        expect(await refusalOf(shop.checkout({plan: "week", target: {kind: "submission", id}, startsNow: true}))).toBe("not-found")
        shop.refuse()
        expect(await refusalOf(shop.checkout({plan: "week", target: {kind: "submission", id}, startsNow: true}))).toBe("sent")
    })
})
