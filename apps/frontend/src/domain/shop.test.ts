import {describe, expect, it} from "vitest"
import type {SoldFlag} from "./flags.ts"
import type {Submission} from "./flagSubmissions.ts"
import {awaitsPayment, canAddTime, COMEBACK_MS, flagRenewal, flagTimeOf, payable, Renewal} from "./shop.ts"

const NOW = 1_000_000_000_000
const PICTURE = {url: "https://media.clickplanet.lol/flags/pictures/ab.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5}

const submission = (changes: Partial<Submission> = {}): Submission => ({
    id: "s-1", name: "Bavaria", change: false, flag: "", state: "approved", submittedAt: 1, decidedAt: 2, ...changes,
})

const renewal = (changes: Partial<Renewal> = {}): Renewal => ({
    id: "r-1", plan: "month", target: {kind: "flag", flag: "r-k3p9x2ma0b"}, state: "renewing", periodEndsAt: NOW + 1, ...changes,
})

const sold = (changes: Partial<SoldFlag> = {}): SoldFlag => ({picture: PICTURE, endsAt: NOW + 5, ...changes})

describe("payable", () => {
    it("is an approved new flag alone", () => {
        expect(payable(submission())).toBe(true)
        expect(payable(submission({change: true, flag: "r-k3p9x2ma0b"}))).toBe(false)
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
