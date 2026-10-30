// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {Finale, FINALE_TEASED_STORAGE_KEY} from "../../domain/battle.ts"
import {FinaleTeaserState, useFinaleTeaser} from "./useFinaleTeaser.ts"

const HOUR = 60 * 60 * 1000
const now = Date.UTC(2026, 9, 30, 18)
const finale: Finale = {roundId: "season-0/finale", startsAt: now + 27 * HOUR, endsAt: now + 29 * HOUR, times: 3, description: "", live: false}
const next: Finale = {...finale, roundId: "season-1/finale"}

let latest: FinaleTeaserState

function Harness({finale, at = now}: {finale?: Finale, at?: number}) {
    latest = useFinaleTeaser(finale, at)
    return null
}

beforeEach(() => {
    window.localStorage.clear()
})

afterEach(cleanup)

describe("useFinaleTeaser", () => {
    it("teases a Final Battle once per browser", () => {
        const {rerender} = render(<Harness finale={finale}/>)
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
        expect(latest.finale).toBeUndefined()
    })

    it("waits for the two days before it, and never teases one that plays", () => {
        const {rerender} = render(<Harness finale={finale} at={finale.startsAt - 49 * HOUR}/>)
        expect(latest.finale).toBeUndefined()

        rerender(<Harness finale={{...finale, live: true}}/>)
        expect(latest.finale).toBeUndefined()
    })
})
