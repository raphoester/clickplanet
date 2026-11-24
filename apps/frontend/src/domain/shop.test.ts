import {describe, expect, it} from "vitest"
import {FlagCatalog, type SoldFlag} from "./flags.ts"
import type {Submission} from "./flagSubmissions.ts"
import {
    awaitsPayment, bestValueOf, canAddTime, COMEBACK_MS, flagRenewal, flagTimeOf, offerOf, payable, pickedOf, PlanPrice,
    Renewal, savingOf, perDayOf,
} from "./shop.ts"

const PLANS: PlanPrice[] = [
    {plan: "week", amount: 400, currency: "eur", renews: false},
    {plan: "month", amount: 1000, currency: "eur", renews: true},
    {plan: "year", amount: 6000, currency: "eur", renews: true},
]
const [WEEK, MONTH, YEAR] = PLANS

const NOW = 1_000_000_000_000
const PICTURE = {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}

const submission = (changes: Partial<Submission> = {}): Submission => ({
    id: "s-1", name: "Bavaria", flag: "", state: "approved", submittedAt: 1, decidedAt: 2, ...changes,
})

const renewal = (changes: Partial<Renewal> = {}): Renewal => ({
    id: "r-1", plan: "month", target: {kind: "flag", flag: "r-k3p9x2ma0b"}, state: "renewing", periodEndsAt: NOW + 1, ...changes,
})

const sold = (changes: Partial<SoldFlag> = {}): SoldFlag => ({picture: PICTURE, endsAt: NOW + 5, ...changes})

describe("payable", () => {
    it("is an approved flag alone", () => {
        expect(payable(submission())).toBe(true)
        expect(payable(submission({state: "opened", flag: "r-k3p9x2ma0b"}))).toBe(false)
        expect(payable(submission({state: "waiting"}))).toBe(false)
    })

    it("tells whether any submission still waits for its payment", () => {
        expect(awaitsPayment([submission({state: "opened"}), submission({id: "s-2", state: "rejected"})])).toBe(false)
        expect(awaitsPayment([submission({id: "s-2", state: "waiting"}), submission()])).toBe(true)
    })
})

describe("flagRenewal", () => {
    it("finds a renewal on the flag, or on the submission that opened it", () => {
        const onFlag = renewal()
        const onSubmission = renewal({id: "r-2", target: {kind: "submission", id: "s-1"}})
        const opened = [submission({state: "opened", flag: "r-k3p9x2ma0b"})]

        expect(flagRenewal([onFlag], "r-k3p9x2ma0b", [])).toBe(onFlag)
        expect(flagRenewal([onSubmission], "r-k3p9x2ma0b", opened)).toBe(onSubmission)
        expect(flagRenewal([onSubmission], "r-0000000000", opened)).toBeUndefined()
    })
})

describe("canAddTime", () => {
    it("takes an open flag, or one closed less than 30 days ago", () => {
        expect(canAddTime(sold(), false, NOW)).toBe(true)
        expect(canAddTime(sold({closedAt: NOW - COMEBACK_MS + 1}), true, NOW)).toBe(true)
        expect(canAddTime(sold({closedAt: NOW - COMEBACK_MS}), true, NOW)).toBe(false)
        expect(canAddTime(sold(), true, NOW)).toBe(false)
        expect(canAddTime(undefined, false, NOW)).toBe(false)
    })
})

describe("flagTimeOf", () => {
    it("says a renewal first, then the time more can be added to", () => {
        expect(flagTimeOf(sold(), false, renewal(), NOW)).toEqual({kind: "renews", renewal: renewal()})
        expect(flagTimeOf(sold(), false, renewal({state: "stopping"}), NOW)).toEqual({kind: "stopping", renewal: renewal({state: "stopping"})})
        expect(flagTimeOf(sold(), false, undefined, NOW)).toEqual({kind: "open", endsAt: NOW + 5})
        expect(flagTimeOf(sold({closedAt: NOW - 1}), true, undefined, NOW)).toEqual({kind: "closed", closedAt: NOW - 1})
        expect(flagTimeOf(sold({closedAt: NOW - COMEBACK_MS}), true, undefined, NOW)).toEqual({kind: "none"})
    })
})

describe("the plans side by side", () => {
    it("spreads each plan over a day, rounded up", () => {
        expect(PLANS.map(perDayOf)).toEqual([58, 33, 17])
        expect(perDayOf({plan: "week", amount: 700, currency: "eur", renews: false})).toBe(100)
    })

    it("says what each plan saves by the day against the dearest, rounded down", () => {
        expect(PLANS.map((price) => savingOf(price, PLANS))).toEqual([0, 42, 71])
        expect([MONTH, YEAR].map((price) => savingOf(price, [MONTH, YEAR]))).toEqual([0, 50])
    })

    it("names the best value only when there is a choice", () => {
        expect(bestValueOf(PLANS)).toBe("year")
        expect(bestValueOf([WEEK, MONTH])).toBe("month")
        expect(bestValueOf([WEEK])).toBeUndefined()
    })

    it("shows the player's pick while it is on sale, else the best value", () => {
        expect(pickedOf(PLANS, undefined)).toBe(YEAR)
        expect(pickedOf(PLANS, "week")).toBe(WEEK)
        expect(pickedOf([WEEK, MONTH], "year")).toBe(MONTH)
        expect(pickedOf([], "week")).toBeUndefined()
    })
})

describe("offerOf", () => {
    const FLAG = "r-k3p9x2ma0b"
    const catalog = (changes: Partial<SoldFlag> = {}, closed = false) =>
        FlagCatalog.BUILT_IN.withFlags([{code: FLAG, name: "Bavaria", closed, sold: sold(changes)}])
    const opened = submission({id: "s-0", state: "opened", flag: FLAG})

    it("sells the opening of an approved flag first", () => {
        expect(offerOf([submission({id: "s-2", state: "waiting"}), submission(), opened], catalog(), [], NOW))
            .toEqual({kind: "open", submission: submission()})
    })

    it("waits for a flag in review", () => {
        const waiting = submission({state: "waiting"})
        expect(offerOf([waiting, opened], catalog(), [], NOW)).toEqual({kind: "waiting", submission: waiting})
    })

    it("sells time for a founded flag, open, closed less than 30 days ago, or whose renewal stops", () => {
        expect(offerOf([opened], catalog(), [], NOW))
            .toEqual({kind: "addTime", flag: FLAG, time: {kind: "open", endsAt: NOW + 5}})
        expect(offerOf([opened], catalog({closedAt: NOW - 1}, true), [], NOW))
            .toEqual({kind: "addTime", flag: FLAG, time: {kind: "closed", closedAt: NOW - 1}})
        expect(offerOf([opened], catalog(), [renewal({state: "stopping"})], NOW))
            .toEqual({kind: "addTime", flag: FLAG, time: {kind: "stopping", renewal: renewal({state: "stopping"})}})
    })

    it("sells no second plan while a renewal runs", () => {
        expect(offerOf([opened], catalog(), [renewal()], NOW))
            .toEqual({kind: "renewing", flag: FLAG, time: {kind: "renews", renewal: renewal()}})
    })

    it("sells a new flag to an account with none to pay for", () => {
        expect(offerOf([], catalog(), [], NOW)).toEqual({kind: "new"})
        expect(offerOf([submission({state: "rejected"})], catalog(), [], NOW)).toEqual({kind: "new"})
        expect(offerOf([opened], catalog({closedAt: NOW - COMEBACK_MS}, true), [], NOW)).toEqual({kind: "new"})
    })
})
