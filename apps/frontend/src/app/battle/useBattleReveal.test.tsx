// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {Battle, BATTLE_SEEN_STORAGE_KEY} from "../../domain/battle.ts"
import {BattleRevealState, useBattleReveal} from "./useBattleReveal.ts"

const battle: Battle = {entryId: "entry-1", since: 1, endsAt: 2, description: "", roundId: "entry-1"}
const later: Battle = {entryId: "entry-2", since: 3, endsAt: 4, description: "", roundId: "entry-2"}

let latest: BattleRevealState

function Harness({battle}: {battle?: Battle}) {
    latest = useBattleReveal(battle)
    return null
}

beforeEach(() => {
    window.localStorage.clear()
})

afterEach(cleanup)

describe("useBattleReveal", () => {
    it("announces a battle once per browser", () => {
        const {rerender} = render(<Harness battle={battle}/>)
        expect(latest.battle).toEqual(battle)

        act(() => latest.dismiss())
        expect(latest.battle).toBeUndefined()
        expect(window.localStorage.getItem(BATTLE_SEEN_STORAGE_KEY)).toBe("entry-1")

        rerender(<Harness battle={later}/>)
        expect(latest.battle).toEqual(later)
    })

    it("remembers across a reload", () => {
        window.localStorage.setItem(BATTLE_SEEN_STORAGE_KEY, "entry-1")
        render(<Harness battle={battle}/>)
        expect(latest.battle).toBeUndefined()
    })

    it("announces nothing with no battle", () => {
        render(<Harness/>)
        expect(latest.battle).toBeUndefined()
    })
})
