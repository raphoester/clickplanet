// @vitest-environment jsdom
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {GetReplayResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {replayFileOf} from "./actions.ts"
import {PaintPanel} from "./PaintPanel.tsx"
import {ReassignPanel} from "./ReassignPanel.tsx"
import {ReplayPanel} from "./ReplayPanel.tsx"

afterEach(cleanup)

function type(name: string, value: string) {
    fireEvent.change(screen.getByRole("textbox", {name}), {target: {value}})
}

describe("ReassignPanel", () => {
    it("reassigns only after a dry run of the same two countries", async () => {
        const admin = new FakeAdmin()
        render(<ReassignPanel planet={admin}/>)
        const reassign = () => screen.getByRole("button", {name: "Reassign"}) as HTMLButtonElement

        type("From", "dz")
        type("To", "FR")
        expect(reassign().disabled).toBe(true)

        fireEvent.click(screen.getByRole("button", {name: "Dry run"}))
        expect(await screen.findByText("Algeria holds 22040 tiles, France 300 tiles.")).toBeTruthy()

        type("To", "es")
        expect(reassign().disabled).toBe(true)
        type("To", "fr")

        fireEvent.click(reassign())
        expect(screen.getByText("Give Algeria's 22040 tiles to France?")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, reassign"}))

        expect(await screen.findByText("Moved 22040 tiles. France holds 22340.")).toBeTruthy()
        expect(admin.askedFor("reassignCountry")).toEqual([
            {fromCountryId: "dz", toCountryId: "fr", dryRun: true},
            {fromCountryId: "dz", toCountryId: "fr", dryRun: false},
        ])
    })

    it("quotes the server's refusal", async () => {
        const admin = new FakeAdmin()
        admin.refuse("reassignCountry", Code.InvalidArgument)
        render(<ReassignPanel planet={admin}/>)

        type("From", "dz")
        type("To", "dz")
        fireEvent.click(screen.getByRole("button", {name: "Dry run"}))

        expect(await screen.findByText("The server refused it: refused")).toBeTruthy()
    })
})

describe("PaintPanel", () => {
    it("paints what it drew on a dry run, on the area asked", async () => {
        const admin = new FakeAdmin()
        render(<PaintPanel planet={admin}/>)

        type("Flag", "dz")
        type("Starting on", "fr")
        fireEvent.change(screen.getByRole("spinbutton", {name: "Tiles"}), {target: {value: "500"}})
        fireEvent.click(screen.getByRole("button", {name: "Dry run"}))
        expect(await screen.findByText(/^4000 tiles of the area do not wear the flag yet/)).toBeTruthy()

        fireEvent.click(screen.getByRole("button", {name: "Paint"}))
        expect(screen.getByText("Paint about 500 tiles with Algeria, from France's ground?")).toBeTruthy()
        fireEvent.click(screen.getByRole("button", {name: "Yes, paint"}))

        expect(await screen.findByText("Painted 498 tiles of 500 picked, 12 past the border.")).toBeTruthy()
        expect(admin.askedFor("paintRandomTiles")).toEqual([
            {flagCountryId: "dz", areaCountryId: "fr", count: 500, proximity: 0.8, dryRun: true},
            {flagCountryId: "dz", areaCountryId: "fr", count: 500, proximity: 0.8, dryRun: false},
        ])
    })
})

describe("ReplayPanel", () => {
    it("fetches the hours asked and saves them as the clip generator reads them", async () => {
        const admin = new FakeAdmin()
        const save = vi.fn()
        render(<ReplayPanel planet={admin} save={save}/>)

        fireEvent.change(screen.getByRole("combobox", {name: "Span"}), {target: {value: "6"}})
        fireEvent.change(screen.getByLabelText("Ending (empty: now)"), {target: {value: "2026-10-18T12:00"}})
        fireEvent.click(screen.getByRole("button", {name: "Download"}))

        const said = await screen.findByRole("status")
        expect(within(said).getByText(/^Saved replay-/)).toBeTruthy()
        const [request] = admin.askedFor("getReplay") as {since: {toDate(): Date}, until: {toDate(): Date}}[]
        const until = new Date("2026-10-18T12:00")
        expect(request.until.toDate()).toEqual(until)
        expect(request.since.toDate()).toEqual(new Date(until.getTime() - 6 * 60 * 60 * 1000))
        expect(save).toHaveBeenCalledTimes(1)
    })
})

describe("replayFileOf", () => {
    it("names the file by its window and keeps the answer as protobuf JSON", () => {
        const answer = new GetReplayResponse({
            since: {seconds: BigInt(AT.getTime() / 1000 - 3600)},
            until: {seconds: BigInt(AT.getTime() / 1000)},
        })

        const file = replayFileOf(answer)
        expect(file.name).toBe("replay-20261018T1100Z-20261018T1200Z.json")
        expect(GetReplayResponse.fromJsonString(file.json)).toEqual(answer)
    })
})
