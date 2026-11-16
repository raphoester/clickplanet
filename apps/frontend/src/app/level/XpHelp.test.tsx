// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, fireEvent, render, screen} from "@testing-library/react"
import {PlayerLevel} from "../../backends/progression.ts"
import XpHelp from "./XpHelp.tsx"

afterEach(cleanup)

const twelve: PlayerLevel = {level: 12, xp: 20_000, levelXp: 18_634, nextLevelXp: 24_192}

const help = () => screen.getByRole("button", {name: "How to gain XP"})

const ways = () => screen.queryByRole("status")

describe("XpHelp", () => {
    it("says nothing until the player asks", () => {
        render(<XpHelp level={twelve}/>)

        expect(ways()).toBeNull()
        expect(help().getAttribute("aria-expanded")).toBe("false")
    })

    it("says what the bar is and how to gain XP on a press, and folds on a second", () => {
        render(<XpHelp level={twelve}/>)

        fireEvent.click(help())

        expect(ways()?.textContent).toBe("XP to level 131,366 / 5,558 XPEach new level gives you a bonusHow to gain XPTake tilesPlay every dayComplete missionsDrop bombsCatch boxesAnswer quizzes rightFortify tiles")
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
})
