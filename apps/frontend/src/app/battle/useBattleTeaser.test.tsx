// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it} from "vitest"
import {act, cleanup, render, waitFor} from "@testing-library/react"
import {FakePromoBackend} from "../../backends/fakePromoBackend.ts"
import {BattleToCome} from "../../backends/turn.ts"
import {BATTLE_TEASED_STORAGE_KEY, BATTLE_TEASER_EVERY_MS} from "../../domain/battleTeaser.ts"
import {acceptedClicks, AcceptedClicks} from "../viewer/acceptedClicks.ts"
import {BattleTeaserState, useBattleTeaser} from "./useBattleTeaser.ts"

const HOUR = 60 * 60 * 1000
const now = Date.UTC(2026, 10, 5, 8)
const battle: BattleToCome = {
    entryId: "tonight", startsAt: Date.UTC(2026, 10, 5, 20), endsAt: Date.UTC(2026, 10, 5, 21),
    description: "", times: 2, endsSeason: false, roundId: "tonight",
}

let latest: BattleTeaserState
let clicks: AcceptedClicks
let promo: FakePromoBackend

function Harness({next = battle, at = now, guided = false}: {next?: BattleToCome, at?: number, guided?: boolean}) {
    latest = useBattleTeaser(next, at, guided, clicks.listenForClicks, promo)
    return null
}

beforeEach(() => {
    window.localStorage.clear()
    clicks = acceptedClicks()
    promo = new FakePromoBackend()
})

afterEach(cleanup)

describe("useBattleTeaser", () => {
    it("teases the next battle with promo's cards", async () => {
        render(<Harness/>)

        await waitFor(() => expect(latest.teaser?.cards).toEqual(promo.cards))
        expect(latest.teaser?.battle).toEqual(battle)
        expect(promo.asked).toEqual(["tonight"])
    })

    it("teases nothing for a battle with no cards, and keeps nothing", async () => {
        promo.cards = []
        render(<Harness/>)

        await waitFor(() => expect(promo.asked).toEqual(["tonight"]))
        expect(latest.teaser).toBeUndefined()
        expect(window.localStorage.getItem(BATTLE_TEASED_STORAGE_KEY)).toBeNull()
    })

    it("asks nothing before the day of the battle", () => {
        render(<Harness at={battle.startsAt - 25 * HOUR}/>)

        expect(latest.teaser).toBeUndefined()
        expect(promo.asked).toEqual([])
    })

    it("tells a browser again once the hours passed, not before", async () => {
        const {rerender} = render(<Harness/>)
        await waitFor(() => expect(latest.teaser).toBeDefined())

        act(() => latest.dismiss(battle))
        expect(latest.teaser).toBeUndefined()
        expect(JSON.parse(window.localStorage.getItem(BATTLE_TEASED_STORAGE_KEY) ?? "")).toMatchObject({key: "tonight"})

        const at = JSON.parse(window.localStorage.getItem(BATTLE_TEASED_STORAGE_KEY) ?? "").at as number
        rerender(<Harness at={at + BATTLE_TEASER_EVERY_MS - 1}/>)
        expect(latest.teaser).toBeUndefined()
        rerender(<Harness at={at + BATTLE_TEASER_EVERY_MS}/>)
        await waitFor(() => expect(latest.teaser).toBeDefined())
    })

    it("waits until a newcomer took its clicks", async () => {
        render(<Harness guided/>)
        expect(promo.asked).toEqual([])

        act(() => {
            for (let i = 0; i < 29; i++) clicks.record({country: "fr", took: true})
        })
        expect(promo.asked).toEqual([])

        act(() => clicks.record({country: "fr", took: true}))
        await waitFor(() => expect(latest.teaser).toBeDefined())
    })

    it("teases the Final Battle by its round", async () => {
        const finale = {...battle, entryId: "", endsSeason: true, roundId: "season-1/finale"}
        render(<Harness next={finale}/>)

        await waitFor(() => expect(latest.teaser?.battle).toEqual(finale))
        expect(promo.asked).toEqual(["season-1/finale"])
    })
})
