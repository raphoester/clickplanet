import {Code, ConnectError} from "@connectrpc/connect"
import {describe, expect, it} from "vitest"
import {ReportOutcome} from "../gen/grpc/support/v1/admin_pb.ts"
import {ban, decide, DEFAULT_MUTE_SECONDS, inspect, LADDER, mute, refusalOf, removeMessage} from "./actions.ts"
import {AT, FakeAdmin} from "./fakeClients.ts"

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

describe("mute", () => {
    it("mutes the account for the seconds asked", async () => {
        const admin = new FakeAdmin()

        expect(await mute(admin, "account-reported", DEFAULT_MUTE_SECONDS)).toEqual({until: AT})
        const [{request}] = admin.asked as {request: {accountId: string, duration: {seconds: bigint}}}[]
        expect(request.accountId).toBe("account-reported")
        expect(request.duration.seconds).toBe(3600n)
    })
})

describe("ban", () => {
    it("leaves the duration to the ladder by default", async () => {
        const admin = new FakeAdmin()

        expect(await ban(admin, "account-reported", LADDER)).toEqual({until: AT, offence: 2, enforced: true})
        expect(admin.asked).toEqual([{method: "banPlayer", request: {accountId: "account-reported", duration: undefined}}])
    })

    it("sends a duration picked", async () => {
        const admin = new FakeAdmin()

        await ban(admin, "account-reported", 86400)
        const [{request}] = admin.asked as {request: {duration: {seconds: bigint}}}[]
        expect(request.duration.seconds).toBe(86400n)
    })

    it("says the antibot is off when the server cannot ban", async () => {
        const admin = new FakeAdmin()
        admin.refuse("banPlayer", Code.FailedPrecondition)

        const err = await ban(admin, "account-reported", LADDER).catch((e: unknown) => e)
        expect(refusalOf("ban", err)).toBe("The antibot is off on this server, so it cannot ban or inspect.")
    })
})

describe("inspect", () => {
    it("inspects the account by its id", async () => {
        const admin = new FakeAdmin()

        expect((await inspect(admin, "account-reported")).clicks).toBe(42)
        expect(admin.asked).toEqual([{method: "inspectPlayer", request: {accountId: "account-reported"}}])
    })

    it("says the antibot is off when the server cannot inspect", async () => {
        const admin = new FakeAdmin()
        admin.refuse("inspectPlayer", Code.FailedPrecondition)

        const err = await inspect(admin, "account-reported").catch((e: unknown) => e)
        expect(refusalOf("inspect", err)).toBe("The antibot is off on this server, so it cannot ban or inspect.")
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

describe("refusalOf", () => {
    it("says the server's reason for a request it refused", () => {
        expect(refusalOf("mute", new ConnectError("a mute names an account", Code.InvalidArgument)))
            .toBe("The server refused it: a mute names an account")
    })

    it("says a session that ended, and an account that is not an operator", () => {
        expect(refusalOf("ban", new ConnectError("x", Code.Unauthenticated)))
            .toBe("Your session ended. Reload the page and sign in again.")
        expect(refusalOf("ban", new ConnectError("x", Code.PermissionDenied))).toBe("This account is not an operator.")
    })

    it("says any other failure with its message", () => {
        expect(refusalOf("mute", new TypeError("Failed to fetch"))).toBe("The call failed: Failed to fetch")
    })
})
