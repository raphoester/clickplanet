// @vitest-environment jsdom
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {Timestamp} from "@bufbuild/protobuf"
import {timeOf} from "../components/time.ts"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {GetReplayResponse, ReplayedEvent} from "../gen/grpc/planet/v1/admin_pb.ts"
import {BoardEra} from "../gen/grpc/planet/v1/history_pb.ts"
import {PlanetEvent, TileUpdate} from "../gen/grpc/planet/v1/planet_pb.ts"
import {boardsOf, erasOf, replayFileOf} from "./actions.ts"
import {SavedFile} from "./download.ts"
import {ReplayPanel} from "./ReplayPanel.tsx"

const MINUTE = 60_000
const SWITCH = new Date(AT.getTime() - 30 * MINUTE)

const minutesBefore = (minutes: number) => Timestamp.fromDate(new Date(AT.getTime() - minutes * MINUTE))

// A take on `tile`, `minutes` before AT.
const takeAt = (minutes: number, tile: number) => new ReplayedEvent({
    at: minutesBefore(minutes),
    event: new PlanetEvent({event: {case: "tileUpdate", value: new TileUpdate({tileId: tile, countryId: "fr"})}}),
})

// The hour before AT: two takes on earth-1, the switch half an hour before AT, then three takes on fixture-1, whose
// tile ids are its own.
function acrossTheSwitch(): GetReplayResponse {
    return new GetReplayResponse({
        since: minutesBefore(60),
        until: Timestamp.fromDate(AT),
        eras: [
            new BoardEra({startedAt: Timestamp.fromDate(new Date(0)), board: "earth-1"}),
            new BoardEra({startedAt: Timestamp.fromDate(SWITCH), board: "fixture-1"}),
        ],
        events: [takeAt(50, 200_000), takeAt(31, 200_001), takeAt(30, 2_998), takeAt(20, 1), takeAt(5, 2)],
    })
}

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

describe("a replay across a switch of board", () => {
    it("says each board the file is on, and when the world switched", async () => {
        const admin = new FakeAdmin()
        admin.replay = acrossTheSwitch()
        const saved: SavedFile[] = []
        render(<ReplayPanel planet={admin} save={file => saved.push(file)}/>)

        fireEvent.click(screen.getByRole("button", {name: "Download"}))

        const said = await screen.findByRole("status")
        expect(said.textContent).toContain(`on earth-1 (2 events), then on fixture-1 from ${timeOf(SWITCH)} (3 events).`)
        const kept = GetReplayResponse.fromJsonString(saved[0].json)
        expect(kept.eras.map(era => [era.board, era.startedAt?.toDate()])).toEqual([["earth-1", new Date(0)], ["fixture-1", SWITCH]])
        expect(kept.events).toHaveLength(5)
    })

    it("puts each event on the era in force at its time, the switch itself the new board's", () => {
        expect(erasOf(acrossTheSwitch()).map(era => [era.board, era.events])).toEqual([["earth-1", 2], ["fixture-1", 3]])
    })

    it("names one board alone, and says nothing of an answer that names none", () => {
        const one = new GetReplayResponse({eras: [new BoardEra({board: "earth-1"})], events: [takeAt(5, 1)]})

        expect(boardsOf(erasOf(one))).toBe("on earth-1")
        expect(boardsOf(erasOf(new GetReplayResponse()))).toBe("")
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

    it("keeps every era and every event, each event's tile ids its era's board's", () => {
        const answer = acrossTheSwitch()

        const file = replayFileOf(answer)
        expect(file.name).toBe("replay-20261018T1100Z-20261018T1200Z.json")
        expect(GetReplayResponse.fromJsonString(file.json).equals(answer)).toBe(true)
        expect(JSON.parse(file.json).eras).toEqual([
            {startedAt: "1970-01-01T00:00:00Z", board: "earth-1"},
            {startedAt: "2026-10-18T11:30:00Z", board: "fixture-1"},
        ])
    })
})
