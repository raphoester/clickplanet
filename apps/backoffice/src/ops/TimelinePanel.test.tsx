// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {TimelinePanel} from "./TimelinePanel.tsx"

afterEach(cleanup)

describe("the timeline page", () => {
    it("reads the schedule, the seasons and planet's sets, and draws a lane of each", async () => {
        const admin = new FakeAdmin()
        render(<TimelinePanel gamemaster={admin.gamemaster} planet={admin}/>)

        const lanes = await screen.findByRole("table", {name: "Blocks"})
        expect(within(lanes).getAllByRole("rowheader").map(header => header.textContent))
            .toEqual(["Seasons", "Rules in force", "Battles", "Days"])
        expect(admin.askedFor("getSchedule")).toEqual([{}])
        expect(admin.askedFor("getSeasons")).toEqual([{}])
        expect(admin.askedFor("getRules")).toEqual([{}])
    })

    it("says when nothing happens in the range", async () => {
        render(<TimelinePanel gamemaster={new FakeAdmin().gamemaster} planet={new FakeAdmin()}/>)

        expect(await screen.findByText("Nothing happens in this range.")).toBeTruthy()
    })

    it("shows the read's failure", async () => {
        const admin = new FakeAdmin()
        admin.refuse("getSeasons", Code.Unavailable)
        render(<TimelinePanel gamemaster={admin.gamemaster} planet={admin}/>)

        expect(await screen.findByRole("button", {name: "Try again"})).toBeTruthy()
    })

    it("marks the range picked", async () => {
        const admin = new FakeAdmin()
        render(<TimelinePanel gamemaster={admin.gamemaster} planet={admin}/>)

        fireEvent.click(await screen.findByRole("button", {name: "4 weeks"}))

        expect(screen.getByRole("button", {name: "4 weeks"}).getAttribute("aria-pressed")).toBe("true")
        expect(screen.getByRole("button", {name: "3 days"}).getAttribute("aria-pressed")).toBe("false")
    })
})
