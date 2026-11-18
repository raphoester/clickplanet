// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import {PlayerLevel} from "../../backends/progression.ts"
import XpHelp from "./XpHelp.tsx"
import {useLevel} from "./levelStore.ts"

afterEach(() => {
    cleanup()
    useLevel.setState(useLevel.getInitialState(), true)
})

const LADDER = {spreadClicks: 2, shields: 3, encloseClicks: 5, bomb: 7, virus: 10, mine: 13, rally: 15}
const RATES = {
    tilesTaken: {xp: 1, per: 1},
    daysPlayed: {xp: 100, per: 1},
    bombsDropped: {xp: 50, per: 1},
    boxesCaught: {xp: 30, per: 1},
    rightAnswers: {xp: 30, per: 1},
    tilesFortified: {xp: 1, per: 5},
}

const twelve: PlayerLevel = {level: 12, xp: 20_000, levelXp: 18_634, nextLevelXp: 24_192}

const help = () => screen.getByRole("button", {name: "How to gain XP"})

const ways = () => screen.queryByRole("status")

describe("XpHelp", () => {
    it("says nothing until the player asks", () => {
        render(<XpHelp level={twelve}/>)

        expect(ways()).toBeNull()
        expect(help().getAttribute("aria-expanded")).toBe("false")
    })

    it("says what the bar is, what the next levels give and what each way to gain XP pays on a press, and folds on a second", () => {
        useLevel.setState({unlocks: LADDER, xpRates: RATES})
        render(<XpHelp level={twelve}/>)

        fireEvent.click(help())

        expect(ways()?.textContent).toBe("XP to level 131,366 / 5,558 XPNext: Mine, unlocks at level 13" +
            "Each level up also gives you a free bonusHow to gain XP" +
            "Take tiles+1 XPFirst take of each day+100 XPComplete missionsShown on eachDrop bombs+50 XP" +
            "Catch boxes+30 XPAnswer quizzes right+30 XPFortify tiles+0.2 XP")
        expect(help().getAttribute("aria-expanded")).toBe("true")

        fireEvent.click(help())

        expect(ways()).toBeNull()
    })

    it("lists them while a mouse rests on the line, and folds when it leaves", () => {
        render(<XpHelp level={twelve}/>)

        fireEvent.pointerEnter(help(), {pointerType: "mouse"})
        expect(ways()).not.toBeNull()

        fireEvent.pointerLeave(help(), {pointerType: "mouse"})
        expect(ways()).toBeNull()
    })

    it("keeps them open after a press, when the mouse leaves", () => {
        render(<XpHelp level={twelve}/>)

        fireEvent.pointerEnter(help(), {pointerType: "mouse"})
        fireEvent.click(help())
        fireEvent.pointerLeave(help(), {pointerType: "mouse"})

        expect(ways()).not.toBeNull()
    })

    it("opens on no touch but a tap", () => {
        render(<XpHelp level={twelve}/>)

        fireEvent.pointerEnter(help(), {pointerType: "touch"})

        expect(ways()).toBeNull()
    })

    it("folds on a press elsewhere and on Escape", () => {
        render(<XpHelp level={twelve}/>)

        fireEvent.click(help())
        fireEvent.pointerDown(document.body)
        expect(ways()).toBeNull()

        fireEvent.click(help())
        fireEvent.keyDown(document, {key: "Escape"})
        expect(ways()).toBeNull()
    })

    it("says every bonus is unlocked past the ladder's top", () => {
        useLevel.setState({unlocks: LADDER, xpRates: RATES})
        render(<XpHelp level={{level: 25, xp: 194_000, levelXp: 193_536, nextLevelXp: 210_014}}/>)

        fireEvent.click(help())

        expect(ways()?.textContent).toContain("Every bonus is unlocked")
        expect(ways()?.textContent).not.toContain("Next:")
    })

    it("names no unlock and no XP it was not told", () => {
        render(<XpHelp level={twelve}/>)

        fireEvent.click(help())

        expect(ways()?.textContent).not.toContain("unlock")
        expect(ways()?.textContent).not.toContain("+1 XP")
        expect(ways()?.textContent).toContain("Shown on each")
    })
})
