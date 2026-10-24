// @vitest-environment jsdom
import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {cleanup, fireEvent, render, screen, within} from "@testing-library/react"
import {afterEach, describe, expect, it, vi} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {Row} from "../gen/grpc/clipstats/v1/admin_pb.ts"
import {ClipStatsPage} from "./ClipStatsPage.tsx"

afterEach(cleanup)

const NOW = new Date(AT.getTime() + 3_600_000)

function rowOf(post: string): Row {
    return new Row({
        takenAt: Timestamp.fromDate(AT), postId: post, platform: "tiktok",
        document: JSON.stringify({postId: post, platform: "tiktok", analytics: {analytics: {views: 1}}}),
    })
}

describe("ClipStatsPage", () => {
    it("says whether TikTok's business lane is live, and lists the snapshots", async () => {
        const admin = new FakeAdmin()
        render(<ClipStatsPage clipStats={admin} now={() => NOW}/>)

        const lanes = await screen.findByRole("region", {name: "TikTok business lane"})
        const lane = within(lanes).getAllByRole("row")[1]
        expect(within(lane).getByText("@clickplanet.lol")).toBeTruthy()
        expect(within(lane).getByText("personal")).toBeTruthy()
        expect(within(lane).getByText("0 of 37")).toBeTruthy()
        expect(within(lane).getByText("refused")).toBeTruthy()
        expect(within(lane).getByText("not live")).toBeTruthy()

        const snapshots = screen.getByRole("region", {name: "Snapshots"})
        expect(within(snapshots).getAllByRole("row")).toHaveLength(3)
        expect(admin.askedFor("listSnapshots")).toEqual([{limit: 14}])
    })

    it("downloads the last snapshot as a CSV, every page of it", async () => {
        const admin = new FakeAdmin()
        admin.rowPages = [[rowOf("post-1"), rowOf("post-2")], [rowOf("post-3")]]
        const save = vi.fn()
        render(<ClipStatsPage clipStats={admin} save={save} now={() => NOW}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Download CSV"}))

        expect(await screen.findByText("Saved clip-stats-20261018T1200Z-20261018T1300Z.csv: 3 rows.")).toBeTruthy()
        const [name, csv, type] = save.mock.calls[0] as [string, string, string]
        expect(name).toBe("clip-stats-20261018T1200Z-20261018T1300Z.csv")
        expect(type).toBe("text/csv")
        const lines = csv.trimEnd().split("\r\n")
        expect(lines).toHaveLength(4)
        expect(lines[0]).toContain("metrics.views")
        const asked = admin.askedFor("listRows") as {since?: Timestamp, page: string}[]
        expect(asked.map(call => call.page)).toEqual(["", "1"])
        expect(asked[0].since?.toDate()).toEqual(AT)
    })

    it("asks for every snapshot of a span", async () => {
        const admin = new FakeAdmin()
        render(<ClipStatsPage clipStats={admin} save={vi.fn()} now={() => NOW}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Everything"}))
        fireEvent.click(screen.getByRole("button", {name: "Download CSV"}))

        expect(await screen.findByText(/^Saved clip-stats-start-/)).toBeTruthy()
        expect((admin.askedFor("listRows")[0] as {since?: Timestamp}).since).toBeUndefined()
    })

    it("offers nothing to download before the first snapshot", async () => {
        const admin = new FakeAdmin()
        admin.snapshots = []
        render(<ClipStatsPage clipStats={admin} now={() => NOW}/>)

        expect(await screen.findByText(/^No snapshot yet/)).toBeTruthy()
        expect((screen.getByRole("button", {name: "Download CSV"}) as HTMLButtonElement).disabled).toBe(true)
    })

    it("quotes a refusal of the rows", async () => {
        const admin = new FakeAdmin()
        admin.refuse("listRows", Code.InvalidArgument)
        render(<ClipStatsPage clipStats={admin} save={vi.fn()} now={() => NOW}/>)

        fireEvent.click(await screen.findByRole("button", {name: "Download CSV"}))

        expect(await screen.findByText("The server refused it: refused")).toBeTruthy()
    })
})
