// @vitest-environment jsdom
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {GetReplayResponse} from "../gen/grpc/planet/v1/admin_pb.ts"
import {replayFileOf} from "./actions.ts"
import {ReplayPanel} from "./ReplayPanel.tsx"

afterEach(cleanup)

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
