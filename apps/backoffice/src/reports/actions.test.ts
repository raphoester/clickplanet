import {Code} from "@connectrpc/connect"
import {describe, expect, it} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {decide, refusalOf, removeMessage} from "./actions.ts"

describe("removeMessage", () => {
    it("removes the message by its id and says when", async () => {
        const admin = new FakeAdmin()

        expect(await removeMessage(admin, "message-1")).toEqual({removedAt: AT})
        expect(admin.asked).toEqual([{method: "removeMessage", request: {messageId: "message-1"}}])
    })

    it("reads a message past the retention as gone", async () => {
        const admin = new FakeAdmin()
        admin.refuse("removeMessage", Code.NotFound)

        const err = await removeMessage(admin, "message-1").catch((e: unknown) => e)
        expect(refusalOf("removeMessage", err)).toBe("This message is no longer kept.")
    })
})

describe("decide", () => {
    it("closes the report with its outcome and a trimmed comment", async () => {
        const admin = new FakeAdmin()

        expect(await decide(admin, "report-1", ReportOutcome.ACTED, "  removed and muted \n"))
            .toEqual({outcome: ReportOutcome.ACTED, comment: "removed and muted", decidedAt: AT})
        expect(admin.asked).toEqual([{
            method: "decideReport",
            request: {reportId: "report-1", outcome: ReportOutcome.ACTED, comment: "removed and muted"},
        }])
    })

    it("reads a report no longer kept as gone", async () => {
        const admin = new FakeAdmin()
        admin.refuse("decideReport", Code.NotFound)

        const err = await decide(admin, "report-1", ReportOutcome.DISMISSED, "").catch((e: unknown) => e)
        expect(refusalOf("decide", err)).toBe("This report is no longer kept.")
    })
})
