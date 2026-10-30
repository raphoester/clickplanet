// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {Finale, FINALE_TEASED_STORAGE_KEY} from "../../domain/battle.ts"
import {FinaleTeaserState, useFinaleTeaser} from "./useFinaleTeaser.ts"
import {acceptedClicks, AcceptedClicks} from "../viewer/acceptedClicks.ts"

const HOUR = 60 * 60 * 1000
const now = Date.UTC(2026, 9, 30, 18)
const finale: Finale = {roundId: "season-0/finale", startsAt: now + 27 * HOUR, endsAt: now + 29 * HOUR, times: 3, description: "", live: false}
const next: Finale = {...finale, roundId: "season-1/finale"}

let latest: FinaleTeaserState
let clicks: AcceptedClicks

function Harness({finale, at = now}: {finale?: Finale, at?: number}) {
    latest = useFinaleTeaser(finale, at, clicks.listenForClicks)
    return null
}

const play = (times: number) => act(() => {
    for (let i = 0; i < times; i++) clicks.record({country: "fr", took: true})
})

beforeEach(() => {
    window.localStorage.clear()
    clicks = acceptedClicks()
})

afterEach(cleanup)

describe("useFinaleTeaser", () => {
    it("waits until the page took its clicks: a newcomer plays first", () => {
        render(<Harness finale={finale}/>)
        expect(latest.finale).toBeUndefined()

        play(29)
        expect(latest.finale).toBeUndefined()

        play(1)
        expect(latest.finale).toEqual(finale)
    })

    it("teases a Final Battle once per browser", () => {
        const {rerender} = render(<Harness finale={finale}/>)
        play(30)
        expect(latest.finale).toEqual(finale)

        act(() => latest.dismiss(finale))
        expect(latest.finale).toBeUndefined()
        expect(window.localStorage.getItem(FINALE_TEASED_STORAGE_KEY)).toBe("season-0/finale")

        rerender(<Harness finale={next}/>)
        expect(latest.finale).toEqual(next)
    })

    it("remembers across a reload", () => {
        window.localStorage.setItem(FINALE_TEASED_STORAGE_KEY, "season-0/finale")
        render(<Harness finale={finale}/>)
        play(30)
        expect(latest.finale).toBeUndefined()
    })

    it("waits for the two days before it, and never teases one that plays", () => {
        const {rerender} = render(<Harness finale={finale} at={finale.startsAt - 49 * HOUR}/>)
        play(30)
        expect(latest.finale).toBeUndefined()

        rerender(<Harness finale={{...finale, live: true}}/>)
        expect(latest.finale).toBeUndefined()
    })
})
