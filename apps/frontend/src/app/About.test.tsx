// @vitest-environment jsdom
import {afterEach, describe, expect, it} from "vitest"
import {cleanup, render, screen} from "@testing-library/react"
import About from "./About.tsx"

afterEach(cleanup)

describe("About", () => {
    it("links to the privacy policy and the terms of service", () => {
        render(<About/>)

        expect(screen.getByRole("link", {name: "Privacy policy"}).getAttribute("href")).toBe("/privacy")
        expect(screen.getByRole("link", {name: "Terms of service"}).getAttribute("href")).toBe("/terms")
    })

    it("links back to the home page, past the redirect to the game", () => {
        render(<About/>)

        expect(screen.getByRole("link", {name: "Home page"}).getAttribute("href")).toBe("/#home")
    })
})
