import {Timestamp} from "@bufbuild/protobuf"
import {Code} from "@connectrpc/connect"
import {describe, expect, it} from "vitest"
import {AT, FakeAdmin} from "../fakeAdmin.ts"
import {GetMuteResponse} from "../gen/grpc/chat/v1/admin_pb.ts"
import {
    ban,
    DEFAULT_MUTE_SECONDS,
    findPlayers,
    grant,
    grantsAnything,
    inspect,
    LADDER,
    mute,
    mutingOf,
    NO_GRANT,
    refusalOf,
    revert,
    topPlayers,
    unban,
    unmute,
} from "./actions.ts"

const account = {accountId: "account-reported"}
const scope = {scope: "203.0.113.7"}

describe("mute", () => {
    it("mutes the account for the seconds asked", async () => {
        const admin = new FakeAdmin()

        expect(await mute(admin, "account-reported", DEFAULT_MUTE_SECONDS)).toEqual({until: AT})
        const [request] = admin.askedFor("mute") as {accountId: string, duration: {seconds: bigint}}[]
        expect(request.accountId).toBe("account-reported")
        expect(request.duration.seconds).toBe(3600n)
    })
})

describe("unmute", () => {
    it("lifts the account's mutes and says until when it was muted", async () => {
        const admin = new FakeAdmin()

        expect(await unmute(admin, "account-reported")).toEqual({until: AT})
        expect(admin.askedFor("unmute")).toEqual([{accountId: "account-reported"}])
    })

    it("says a player with no mute of its own", async () => {
        const admin = new FakeAdmin()
        admin.refuse("unmute", Code.NotFound)

        const err = await unmute(admin, "account-reported").catch((e: unknown) => e)
        expect(refusalOf("unmute", err)).toBe("This player has no mute of its own to lift.")
    })
})

describe("mutingOf", () => {
    it("reads no mute as none", async () => {
        expect(await mutingOf(new FakeAdmin(), "account-reported")).toBeUndefined()
    })

    it("tells the account's own mute from one on its network", async () => {
        const admin = new FakeAdmin()

        admin.muting = new GetMuteResponse({mutedUntil: Timestamp.fromDate(AT), mutedAccountId: "account-reported"})
        expect(await mutingOf(admin, "account-reported")).toEqual({until: AT, byNetwork: false})

        admin.muting = new GetMuteResponse({mutedUntil: Timestamp.fromDate(AT), mutedAccountId: "account-household"})
        expect(await mutingOf(admin, "account-reported")).toEqual({until: AT, byNetwork: true})
    })
})

describe("ban", () => {
    it("leaves the duration to the ladder by default", async () => {
        const admin = new FakeAdmin()

        expect(await ban(admin.antibot, account, LADDER)).toEqual({until: AT, offence: 2, enforced: true})
        expect(admin.asked).toEqual([{method: "banPlayer", request: {accountId: "account-reported", duration: undefined}}])
    })

    it("sends a duration picked", async () => {
        const admin = new FakeAdmin()

        await ban(admin.antibot, account, 86400)
        const [request] = admin.askedFor("banPlayer") as {duration: {seconds: bigint}}[]
        expect(request.duration.seconds).toBe(86400n)
    })

    it("bans an address as its scope, with no account", async () => {
        const admin = new FakeAdmin()

        await ban(admin.antibot, scope, LADDER)
        expect(admin.askedFor("banPlayer")).toEqual([{scope: "203.0.113.7", duration: undefined}])
    })

    it("says the antibot is off when the server cannot ban", async () => {
        const admin = new FakeAdmin()
        admin.refuse("banPlayer", Code.Unimplemented)

        const err = await ban(admin.antibot, account, LADDER).catch((e: unknown) => e)
        expect(refusalOf("ban", err)).toBe("The antibot is off on this server, so it cannot ban, unban or inspect.")
    })
})

describe("unban", () => {
    it("lifts the latest ban and says which", async () => {
        const admin = new FakeAdmin()

        expect(await unban(admin.antibot, scope)).toEqual({offence: 2, until: AT, pardonedUntil: AT})
        expect(admin.askedFor("unbanPlayer")).toEqual([{scope: "203.0.113.7"}])
    })

    it("says a player with no ban left", async () => {
        const admin = new FakeAdmin()
        admin.refuse("unbanPlayer", Code.NotFound)

        const err = await unban(admin.antibot, account).catch((e: unknown) => e)
        expect(refusalOf("unban", err)).toBe("This player has no ban left to lift.")
    })
})

describe("revert", () => {
    it("counts on a dry run and restores on the real one", async () => {
        const admin = new FakeAdmin()

        expect(await revert(admin, account, true)).toEqual({touched: 900, held: 120, restored: 0})
        expect(await revert(admin, account, false)).toEqual({touched: 900, held: 120, restored: 118})
        expect(admin.askedFor("revertPlayer")).toEqual([
            {accountId: "account-reported", dryRun: true},
            {accountId: "account-reported", dryRun: false},
        ])
    })
})

describe("inspect", () => {
    it("inspects the account by its id", async () => {
        const admin = new FakeAdmin()

        expect((await inspect(admin.antibot, account)).clicks).toBe(42)
        expect(admin.asked).toEqual([{method: "inspectPlayer", request: {accountId: "account-reported"}}])
    })

    it("says the antibot is off when the server cannot inspect", async () => {
        const admin = new FakeAdmin()
        admin.refuse("inspectPlayer", Code.Unimplemented)

        const err = await inspect(admin.antibot, account).catch((e: unknown) => e)
        expect(refusalOf("inspect", err)).toBe("The antibot is off on this server, so it cannot ban, unban or inspect.")
    })
})

describe("grant", () => {
    it("sends the charges picked and answers what the account held before and after", async () => {
        const admin = new FakeAdmin()

        const granted = await grant(admin, "account-reported", {...NO_GRANT, bomb: true, enclosures: 3})
        expect(granted.before.enclosures).toBe(2)
        expect(granted.after.bomb).toBe(true)
        expect(admin.askedFor("grantCharges")).toEqual([{
            accountId: "account-reported", refill: false, bomb: true, enclosures: 3, spreadClicks: 0, shields: 0,
        }])
    })

    it("grants nothing until a charge is picked", () => {
        expect(grantsAnything(NO_GRANT)).toBe(false)
        expect(grantsAnything({...NO_GRANT, shields: 1})).toBe(true)
        expect(grantsAnything({...NO_GRANT, refill: true})).toBe(true)
    })
})

describe("the lists", () => {
    it("asks for the top players with the limit picked", async () => {
        const admin = new FakeAdmin()

        const top = await topPlayers(admin, 50)
        expect(top.total).toBe(75)
        expect(top.players).toHaveLength(2)
        expect(admin.askedFor("topPlayers")).toEqual([{limit: 50}])
    })

    it("finds the players of a flag on an area's ground", async () => {
        const admin = new FakeAdmin()

        await findPlayers(admin, "ps", "il", 20)
        expect(admin.askedFor("findPlayers")).toEqual([{flagCountryId: "ps", areaCountryId: "il", limit: 20}])
    })
})
