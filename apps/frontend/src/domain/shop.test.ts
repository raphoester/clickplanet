import {describe, expect, it} from "vitest"
import type {SoldFlag} from "./flags.ts"
import type {Submission} from "./flagSubmissions.ts"
import {
    awaitsPayment, bestValueOf, flagRenewal, flagTimeOf, payable, pickedOf, PlanPrice, Renewal, salePriceOf, saleRuns, savingOf, perDayOf,
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
    id: "s-1", name: "Bavaria", flag: "", state: "approved", submittedAt: 1, decidedAt: 2, paid: false, ...changes,
})

const renewal = (changes: Partial<Renewal> = {}): Renewal => ({
    id: "r-1", plan: "month", target: {kind: "flag", flag: "r-k3p9x2ma0b"}, state: "renewing", periodEndsAt: NOW + 1, ...changes,
})

const sold = (changes: Partial<SoldFlag> = {}): SoldFlag => ({picture: PICTURE, endsAt: NOW + 5, ...changes})

describe("payable", () => {
    it("is a new flag not paid yet, sent or approved", () => {
        expect(payable(submission({state: "waiting"}))).toBe(true)
        expect(payable(submission())).toBe(true)
        expect(payable(submission({state: "waiting", paid: true}))).toBe(false)
        expect(payable(submission({state: "waiting", appealed: true})), "paid once a person approves it").toBe(false)
        expect(payable(submission({state: "approved", appealed: true}))).toBe(true)
        expect(payable(submission({paid: true}))).toBe(false)
        expect(payable(submission({state: "opened", flag: "r-k3p9x2ma0b"}))).toBe(false)
        expect(payable(submission({state: "rejected"}))).toBe(false)
    })

    it("tells whether any submission still waits for its payment", () => {
        expect(awaitsPayment([submission({state: "opened"}), submission({id: "s-2", state: "rejected"})])).toBe(false)
        expect(awaitsPayment([submission({id: "s-2", state: "waiting", paid: true})])).toBe(false)
        expect(awaitsPayment([submission({id: "s-2", state: "waiting"})])).toBe(true)
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

describe("flagTimeOf", () => {
    it("says a removal first, then a renewal, then the time more can be added to, however long ago it closed", () => {
        expect(flagTimeOf(sold(), false, renewal())).toEqual({kind: "renews", renewal: renewal()})
        expect(flagTimeOf(sold(), false, renewal({state: "stopping"}))).toEqual({kind: "stopping", renewal: renewal({state: "stopping"})})
        expect(flagTimeOf(sold(), false, undefined)).toEqual({kind: "open", endsAt: NOW + 5})
        expect(flagTimeOf(sold({closedAt: NOW - 1}), true, undefined)).toEqual({kind: "closed", closedAt: NOW - 1})
        expect(flagTimeOf(sold({closedAt: 1}), true, undefined)).toEqual({kind: "closed", closedAt: 1})
        expect(flagTimeOf(sold({closedAt: NOW - 1, removed: true}), true, renewal())).toEqual({kind: "removed", removedAt: NOW - 1})
        expect(flagTimeOf(undefined, false, undefined)).toEqual({kind: "none"})
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

describe("the sale", () => {
    const sale = {percentOff: 50, endsAt: NOW + 1000}

    it("takes its cut off each plan's price, whole for half of 400, 1000 and 6000", () => {
        expect(PLANS.map((price) => salePriceOf(price, sale))).toEqual([
            {...WEEK, amount: 200},
            {...MONTH, amount: 500},
            {...YEAR, amount: 3000},
        ])
        expect(salePriceOf({...WEEK, amount: 999}, {percentOff: 30, endsAt: NOW}).amount).toBe(699)
    })

    it("runs until its end, its end excluded", () => {
        expect(saleRuns(sale, NOW)).toBe(true)
        expect(saleRuns(sale, sale.endsAt - 1)).toBe(true)
        expect(saleRuns(sale, sale.endsAt)).toBe(false)
        expect(saleRuns(undefined, NOW)).toBe(false)
    })
})
