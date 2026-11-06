import {describe, expect, it} from "vitest"
import {FakeAdmin} from "../fakeAdmin.ts"
import {ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {close, Target, Verdict} from "./verdict.ts"

const target: Target = {accountId: "account-reported", messageId: "message-1", who: "Ana", why: "A report: Chat abuse"}

describe("close", () => {
    it("keeps the operator's comment as the reason of the removal and the mute", async () => {
        const admin = new FakeAdmin()
        const verdict: Verdict = {remove: true, muteFor: 3600, banFor: undefined, comment: " a slur, 2nd offence "}

        await close(admin.clients(), target, verdict, ReportOutcome.ACTED, ["report-1"])

        expect((admin.askedFor("removeMessage") as {reason: string}[])[0].reason).toBe("a slur, 2nd offence")
        expect((admin.askedFor("mute") as {reason: string}[])[0].reason).toBe("a slur, 2nd offence")
    })

    it("keeps the report's reason when the operator wrote no comment", async () => {
        const admin = new FakeAdmin()
        const verdict: Verdict = {remove: true, muteFor: undefined, banFor: undefined, comment: ""}

        await close(admin.clients(), target, verdict, ReportOutcome.ACTED, ["report-1"])

        expect((admin.askedFor("removeMessage") as {reason: string}[])[0].reason).toBe("A report: Chat abuse")
    })
})
