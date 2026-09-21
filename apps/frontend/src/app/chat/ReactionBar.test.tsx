// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {Reaction, ReactionCount} from "../../backends/chat.ts"
import ReactionBar from "./ReactionBar.tsx"

afterEach(cleanup)

const clown = (count: number, reactors: string[], mine = false): ReactionCount =>
    ({reaction: Reaction.CLOWN, count, mine, reactors})

function show(reactions: ReactionCount[]) {
    render(<ReactionBar messageId="m1"
                        reactions={reactions}
                        onReact={vi.fn()}
                        picking={false}
                        setPicking={vi.fn()}/>)
    return screen.getByRole("button", {name: `Clown: ${reactions[0].count}`})
}

/** The popup only exists while the pointer is on the chip, so every case opens it first. */
async function hover(chip: HTMLElement): Promise<HTMLElement> {
    await userEvent.hover(chip)
    return screen.getByRole("tooltip")
}

describe("ReactionBar", () => {
    it("says nothing until the pointer is on a reaction", async () => {
        const chip = show([clown(2, ["Ana", "Bo"])])

        expect(screen.queryByRole("tooltip")).toBeNull()

        await userEvent.hover(chip)
        expect(screen.getByRole("tooltip")).toBeDefined()

        await userEvent.unhover(chip)
        expect(screen.queryByRole("tooltip")).toBeNull()
    })

    it("names everyone who reacted, oldest first, under the reaction", async () => {
        const popup = await hover(show([clown(2, ["Ana", "Bo"])]))

        expect(popup.textContent).toBe("ClownAnaBo")
    })

    it("counts the ones it cannot name after the ones it can", async () => {
        const popup = await hover(show([clown(25, ["Ana", "Bo"])]))

        expect(popup.textContent).toContain("and 23 more")
    })

    it("counts them alone when it can name nobody", async () => {
        expect((await hover(show([clown(3, [])]))).textContent).toContain("3 players")

        cleanup()
        expect((await hover(show([clown(1, [])]))).textContent).toContain("1 player")
    })

    it("lets the count decide, so a name past it is not shown", async () => {
        const popup = await hover(show([clown(1, ["Ana", "Bo"])]))

        expect(popup.textContent).toBe("ClownAna")
    })

    it("describes the chip by the popup, so a screen reader reads who reacted", async () => {
        const chip = show([clown(2, ["Ana", "Bo"])])

        expect(chip.getAttribute("aria-describedby")).toBeNull()

        const popup = await hover(chip)
        expect(chip.getAttribute("aria-describedby")).toBe(popup.id)
    })

    it("opens on the focus too, for a player who does not point", async () => {
        show([clown(2, ["Ana", "Bo"])])

        await userEvent.tab()

        expect(screen.getByRole("tooltip").textContent).toContain("Ana")
    })

    it("still reacts when it is clicked", async () => {
        const onReact = vi.fn()
        render(<ReactionBar messageId="m1"
                            reactions={[clown(1, ["Bo"])]}
                            onReact={onReact}
                            picking={false}
                            setPicking={vi.fn()}/>)

        await userEvent.click(screen.getByRole("button", {name: "Clown: 1"}))

        expect(onReact).toHaveBeenCalledWith(Reaction.CLOWN, true)
    })
})
