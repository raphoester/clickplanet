// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {StrictMode} from "react"
import {afterEach, describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {ArrivalCount, ArrivalSource} from "../gen/grpc/auth/v1/admin_pb.ts"
import {StatsRow} from "../gen/grpc/publishing/v1/admin_pb.ts"
import {LabPage} from "./LabPage.tsx"

afterEach(cleanup)

const NOW = new Date("2026-11-23T10:00:00Z")

function rowOf(postId: string, platform: string, experiment: string, publishedAt: string, metrics: Record<string, unknown>): StatsRow {
    return new StatsRow({
        takenAt: Timestamp.fromDate(NOW), postId, platform,
        document: JSON.stringify({
            postId, platform, publishedAt, ageHours: (NOW.getTime() - Date.parse(publishedAt)) / 3_600_000,
            url: `https://example.test/${postId}`, analytics: {analytics: metrics},
            metadata: {record: {post: {experiment}}},
        }),
    })
}

function lab(): FakeAdmin {
    const admin = new FakeAdmin()
    admin.rowPages = [[
        rowOf("c1", "tiktok", "carousel", "2026-11-21T16:00:00Z",
            {views: 524, follows: 2, saves: 1, completionRate: 0.1609, igReelsAvgWatchTime: 7390, retentionCurve: {}}),
        rowOf("k1", "tiktok", "control", "2026-11-21T18:45:00Z",
            {views: 541, saves: 1, completionRate: 0.2159, igReelsAvgWatchTime: 6590, retentionCurve: {"2": 0.65}}),
        rowOf("f1", "youtube", "far-invasion", "2026-11-20T19:46:00Z", {views: 12_617, likes: 12}),
        rowOf("f2", "youtube", "far-invasion", "2026-11-22T13:48:00Z", {views: 1_262, likes: 9}),
        rowOf("k2", "youtube", "control", "2026-11-21T09:46:00Z", {views: 1_673, likes: 10}),
        rowOf("k3", "youtube", "control", "2026-11-23T08:00:00Z", {views: 300, likes: 1}),
    ]]
    admin.arrivals = [
        new ArrivalCount({day: "2026-11-21", source: ArrivalSource.TIKTOK, accounts: 5}),
        new ArrivalCount({day: "2026-11-22", source: ArrivalSource.YOUTUBE, accounts: 9}),
        new ArrivalCount({day: "2026-11-22", source: ArrivalSource.RECRUIT, accounts: 30}),
    ]
    return admin
}

function cellsOf(region: HTMLElement, name: string): string[] {
    const row = within(region).getAllByRole("row").find(each => within(each).queryByRole("rowheader", {name}) !== null)
    return within(row!).getAllByRole("cell").map(cell => cell.textContent ?? "")
}

describe("LabPage", () => {
    it("sets each experiment against the control of its days, on each platform", async () => {
        const admin = lab()
        render(<StrictMode><LabPage publishing={admin} auth={admin} now={() => NOW}/></StrictMode>)

        const tiktok = await screen.findByRole("region", {name: "TikTok"})
        expect(cellsOf(tiktok, "carousel")).toEqual(["21 Nov", "1", "524", "524", "0", "—", "16%", "7.4 s", "0.0", "0.0", "3.8", "1.9"])
        expect(cellsOf(tiktok, "control")).toEqual(["1", "541", "541", "0", "65%", "22%", "6.6 s", "0.0", "0.0", "0.0", "1.8"])

        const youtube = screen.getByRole("region", {name: "YouTube"})
        expect(cellsOf(youtube, "far-invasion")).toEqual(["20–22 Nov", "2", "6,940", "12,617", "1", "1.5"])
        expect(cellsOf(youtube, "control")).toEqual(["1", "1,673", "1,673", "0", "6.0"])
    })

    it("counts each day's arrivals by platform, the newest first", async () => {
        const admin = lab()
        render(<LabPage publishing={admin} auth={admin} now={() => NOW}/>)

        const arrivals = await screen.findByRole("region", {name: "Arrivals"})
        await within(arrivals).findByText("23 Nov")
        expect(within(arrivals).getAllByRole("row").slice(1).map(row => row.textContent)).toEqual([
            "23 Nov00", "22 Nov09", "21 Nov50", "20 Nov00",
        ])
        const asked = admin.askedFor("countArrivals") as {since: Timestamp}[]
        expect(asked.map(call => call.since.toDate().toISOString())).toEqual(["2026-11-20T00:00:00.000Z"])
        const read = admin.askedFor("listRows") as {since: Timestamp}[]
        expect(read[0].since.toDate().toISOString()).toBe("2026-11-20T00:00:00.000Z")
    })

    it("reads two weeks back once the lab is older", async () => {
        const admin = lab()
        render(<LabPage publishing={admin} auth={admin} now={() => new Date("2026-12-20T10:00:00Z")}/>)

        await screen.findByRole("region", {name: "TikTok"})
        expect((admin.askedFor("listRows")[0] as {since: Timestamp}).since.toDate().toISOString()).toBe("2026-12-06T00:00:00.000Z")
    })

    it("shows the posts when the arrivals cannot be counted", async () => {
        const admin = lab()
        admin.refuse("countArrivals", Code.Unimplemented)
        render(<LabPage publishing={admin} auth={admin} now={() => NOW}/>)

        expect(await screen.findByRole("region", {name: "YouTube"})).toBeTruthy()
        const arrivals = screen.getByRole("region", {name: "Arrivals"})
        expect(await within(arrivals).findByRole("button", {name: "Try again"})).toBeTruthy()
    })

    it("reads again from the day picked", async () => {
        const admin = lab()
        render(<LabPage publishing={admin} auth={admin} now={() => NOW}/>)
        await screen.findByRole("region", {name: "TikTok"})

        fireEvent.change(screen.getByLabelText("Posts since (UTC)"), {target: {value: "2026-11-22"}})

        const youtube = await screen.findByRole("region", {name: "YouTube"})
        expect(cellsOf(youtube, "far-invasion")).toEqual(["22 Nov", "1", "1,262", "1,262", "0", "7.1"])
        expect(within(screen.getByRole("region", {name: "TikTok"})).getByText("No lab post.")).toBeTruthy()
    })
})
