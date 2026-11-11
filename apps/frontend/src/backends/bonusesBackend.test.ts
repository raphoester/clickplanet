import {describe, expect, it} from "vitest"
import {catchOf, grantRulesChanged, offerOf, quizOf} from "./bonusesBackend.ts"
import {BonusEvent, BonusOffered, BonusTaken, Heartbeat, QuizOffered, RulesChanged} from "../gen/grpc/bonuses/v1/bonuses_pb.ts"
import {BonusKind} from "../gen/grpc/planet/v1/planet_pb.ts"

const heartbeat = () => new BonusEvent({event: {case: "heartbeat", value: new Heartbeat()}})

describe("offerOf", () => {
    const offered = (fields: {
        token?: string,
        seed?: number,
        kind?: BonusKind,
        expiresAtUnixMs?: bigint,
    } = {}) => new BonusEvent({
        event: {
            case: "bonusOffered",
            value: new BonusOffered({
                token: "a-token",
                seed: 42,
                kind: BonusKind.REFILL,
                expiresAtUnixMs: 1_000_000n,
                ...fields,
            }),
        },
    })

    it("reads the box the server addressed to this client", () => {
        const offer = offerOf(offered())

        expect(offer?.token).toBe("a-token")
        expect(offer?.seed).toBe(42)
        expect(offer?.reward).toEqual({kind: "refill"})
    })

    it("builds the deadline from how long is left, not from the server's clock", () => {
        const offer = offerOf(offered({expiresAtUnixMs: 1_015_000n}), 5_000, 1_000_000)

        expect(offer?.expiresAt).toBe(5_000 + 15_000)
    })

    it("is unmoved by a client clock that is minutes out", () => {
        const skewed = offerOf(offered({expiresAtUnixMs: 1_015_000n}), 5_000, 1_000_000 + 600_000)
        const honest = offerOf(offered({expiresAtUnixMs: 1_015_000n}), 5_000, 1_000_000)

        expect(honest!.expiresAt - 5_000).toBe(15_000)
        expect(skewed!.expiresAt).toBeLessThan(honest!.expiresAt)
    })

    it("reads a bomb box as one, with no blast radius until it is claimed", () => {
        expect(offerOf(offered({kind: BonusKind.BOMB}))?.reward)
            .toEqual({kind: "bomb", radius: 0})
    })

    it("reads an enclose box as one", () => {
        expect(offerOf(offered({kind: BonusKind.ENCLOSE_CLICKS}))?.reward.kind).toBe("encloseClicks")
    })

    it("reads a shields box as one", () => {
        expect(offerOf(offered({kind: BonusKind.SHIELDS}))?.reward.kind).toBe("shields")
    })

    it("reads a spread box as one", () => {
        expect(offerOf(offered({kind: BonusKind.SPREAD_CLICKS}))?.reward.kind)
            .toBe("spreadClicks")
    })

    it("drops a kind this build cannot describe rather than guessing at it", () => {
        expect(offerOf(offered({kind: BonusKind.UNSPECIFIED}))).toBeUndefined()
    })

    it("drops everything that is not an offer", () => {
        expect(offerOf(heartbeat())).toBeUndefined()
        expect(offerOf(new BonusEvent())).toBeUndefined()
    })
})

describe("quizOf", () => {
    const asked = (fields: {token?: string, expiresAtUnixMs?: bigint} = {}) =>
        new BonusEvent({
            event: {
                case: "quizOffered",
                value: new QuizOffered({token: "a-token", expiresAtUnixMs: 1_000_000n, ...fields}),
            },
        })

    it("reads the banner the server addressed to this client", () => {
        expect(quizOf(asked())).toEqual({token: "a-token", expiresAt: expect.any(Number)})
    })

    it("is a token and a deadline, and nothing about the question at all", () => {
        expect(Object.keys(quizOf(asked())!)).toEqual(["token", "expiresAt"])
    })

    it("builds the deadline from how long is left, not from the server's clock", () => {
        expect(quizOf(asked({expiresAtUnixMs: 1_025_000n}), 5_000, 1_000_000)?.expiresAt).toBe(5_000 + 25_000)
    })

    it("drops everything that is not a banner", () => {
        expect(quizOf(heartbeat())).toBeUndefined()
        expect(quizOf(new BonusEvent({event: {case: "bonusTaken", value: new BonusTaken({countryId: "fr"})}}))).toBeUndefined()
    })
})

describe("catchOf", () => {
    it("reads who caught one", () => {
        const event = new BonusEvent({
            event: {case: "bonusTaken", value: new BonusTaken({countryId: "jp"})},
        })

        expect(catchOf(event)).toEqual({countryId: "jp", quizSubject: undefined})
    })

    it("says when the charge was won by answering a question, and what about", () => {
        const event = new BonusEvent({
            event: {case: "bonusTaken", value: new BonusTaken({countryId: "jp", quizSubjectCountryId: "ee"})},
        })

        expect(catchOf(event)).toEqual({countryId: "jp", quizSubject: "ee"})
    })

    it("drops everything that is not a catch", () => {
        expect(catchOf(heartbeat())).toBeUndefined()
    })
})

describe("grantRulesChanged", () => {
    it("says when the offers took another set", () => {
        expect(grantRulesChanged(new BonusEvent({event: {case: "rulesChanged", value: new RulesChanged()}}))).toBe(true)
    })

    it("says nothing of a heartbeat, or of a case this build does not know", () => {
        expect(grantRulesChanged(heartbeat())).toBe(false)
        expect(grantRulesChanged(new BonusEvent())).toBe(false)
    })
})
