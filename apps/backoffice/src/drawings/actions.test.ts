import {Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {describe, expect, it} from "vitest"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {PlaceReport, ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {
    beforeOf,
    clearArea,
    dateOfInput,
    decidePlace,
    localInputOf,
    mapLinkOf,
    OLDER_THAN_THE_LEDGER,
    paintersOf,
    PLACE_REPORT_GONE,
    refusalOf,
    restoreArea,
} from "./actions.ts"

describe("drawing actions", () => {
    it("links the game close up on the tile", () => {
        expect(mapLinkOf(1234)).toBe("https://clickplanet.lol/?tile=1234")
    })

    it("starts the restore an hour before the report", () => {
        const report = new PlaceReport({createdAt: Timestamp.fromDate(AT)})
        expect(beforeOf(report)).toEqual(new Date(AT.getTime() - 3_600_000))
    })

    it("restores the area as it was, and answers what changed", async () => {
        const admin = new FakeAdmin()
        const before = new Date(AT.getTime() - 3_600_000)

        const done = await restoreArea(admin, 1234, 8, before, false)

        expect(admin.askedFor("restoreArea")).toEqual([{tileId: 1234, rings: 8, before: Timestamp.fromDate(before), dryRun: false}])
        expect(done).toMatchObject({tiles: 37, changed: 12, restored: 12, paintersTotal: 2})
        expect(done.keptSince).toEqual(admin.keptSince)
    })

    it("names the painters of a dry run, each account once", async () => {
        const admin = new FakeAdmin()

        const counted = await paintersOf(admin, admin, 1234, 8, new Date(AT.getTime() - 3_600_000))

        expect(admin.askedFor("restoreArea")).toEqual([expect.objectContaining({dryRun: true})])
        expect(admin.askedFor("getAuthors")).toEqual([{accountIds: [ACCOUNT]}])
        expect(counted.authors.get(ACCOUNT)?.name).toBe("BraveFox42")
        expect(counted.unnamed).toBeUndefined()
    })

    it("clears the area, and answers how many tiles it gave to nobody", async () => {
        const admin = new FakeAdmin()

        expect(await clearArea(admin, 1234, 8, true)).toEqual({tiles: 37, owned: 20, cleared: 0})
        expect(await clearArea(admin, 1234, 8, false)).toEqual({tiles: 37, owned: 20, cleared: 20})
        expect(admin.askedFor("clearArea")).toEqual([
            {tileId: 1234, rings: 8, dryRun: true},
            {tileId: 1234, rings: 8, dryRun: false},
        ])
    })

    it("closes a place report with its outcome and a trimmed comment", async () => {
        const admin = new FakeAdmin()

        expect(await decidePlace(admin, "place-report-1", ReportOutcome.ACTED, " restored \n"))
            .toEqual({outcome: ReportOutcome.ACTED, comment: "restored", decidedAt: AT})
        expect(admin.askedFor("decidePlaceReport"))
            .toEqual([{reportId: "place-report-1", outcome: ReportOutcome.ACTED, comment: "restored"}])
    })

    it("says each refusal in the operator's words", () => {
        expect(refusalOf("restore", new ConnectError("x", Code.FailedPrecondition))).toBe(OLDER_THAN_THE_LEDGER)
        expect(refusalOf("decide", new ConnectError("x", Code.NotFound))).toBe(PLACE_REPORT_GONE)
        expect(refusalOf("clear", new ConnectError("rings is 1 to 50", Code.InvalidArgument)))
            .toBe("The server refused it: rings is 1 to 50")
    })

    it("reads back the time a datetime-local input holds, to the minute", () => {
        const at = new Date(2026, 9, 18, 13, 5)
        expect(dateOfInput(localInputOf(at))).toEqual(at)
        expect(dateOfInput("")).toBeUndefined()
    })
})
