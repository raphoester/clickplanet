// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render} from "@testing-library/react"
import TitleEmblem from "./TitleEmblem.tsx"

afterEach(cleanup)

describe("TitleEmblem", () => {
    it("draws the first letter of a title this build has no picture for", () => {
        const {container} = render(<TitleEmblem title={{id: "grand_warmaster", name: "Grand Warmaster"}} size={56}/>)

        expect(container.querySelector("text")?.textContent).toBe("G")
    })

    it("gives each medal its own gradient, so two on one page do not share one", () => {
        const {container} = render(<>
            <TitleEmblem title={{id: "settler", name: "Settler"}} size={56}/>
            <TitleEmblem title={{id: "raider", name: "Raider"}} size={56}/>
        </>)

        const ids = [...container.querySelectorAll("linearGradient")].map((gradient) => gradient.id)
        expect(new Set(ids).size).toBe(2)
    })
})
