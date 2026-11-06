import {Duration, Timestamp} from "@bufbuild/protobuf"
import {describe, expect, it} from "vitest"
import {ACCOUNT, AT, FakeAdmin} from "../fakeAdmin.ts"
import {ModerationAction, MuteAction, RemovalAction, UnmuteAction} from "../gen/grpc/chat/v1/admin_pb.ts"
import {Gif, MuteKind} from "../gen/grpc/chat/v1/chat_pb.ts"
import {actorOf, actorWords, historyOf} from "./history.ts"

const minutesAgo = (minutes: number) => Timestamp.fromDate(new Date(AT.getTime() - minutes * 60_000))

describe("historyOf", () => {
    it("reads the chat's actions and the renames of one account, newest first", async () => {
        const admin = new FakeAdmin()

        const lines = await historyOf(admin, admin, ACCOUNT)

        expect(admin.askedFor("listModeration")).toEqual([{accountId: ACCOUNT, limit: 500}])
        expect(admin.askedFor("listRenames")).toEqual([{accountId: ACCOUNT, limit: 500}])
        expect(lines.map(line => [line.what, line.reason, actorWords(line.by)])).toEqual([
            ["GIFs muted for 15 minutes", "a gore GIF, 2nd offence", "Claude"],
            ["Renamed from Hitler88 to Player42", "a Nazi reference", "Operator"],
        ])
    })

    it("says what each kind of action did, and keeps what a removed message said", async () => {
        const admin = new FakeAdmin()
        admin.renames = []
        admin.moderation = [
            new ModerationAction({
                actionId: "4", actedAt: minutesAgo(1),
                action: {case: "removal", value: new RemovalAction({messageId: "m-2", body: {case: "gif", value: new Gif({url: "https://static.klipy.com/a.webp"})}})},
            }),
            new ModerationAction({
                actionId: "3", actedAt: minutesAgo(2),
                action: {case: "removal", value: new RemovalAction({messageId: "m-1", body: {case: "text", value: "you all suck"}})},
            }),
            new ModerationAction({
                actionId: "2", actedAt: minutesAgo(3),
                action: {case: "unmute", value: new UnmuteAction({kind: MuteKind.CHAT})},
            }),
            new ModerationAction({
                actionId: "1", actedAt: minutesAgo(4),
                action: {case: "mute", value: new MuteAction({kind: MuteKind.CHAT, duration: new Duration({seconds: 9000n})})},
            }),
        ]

        const lines = await historyOf(admin, admin, ACCOUNT)

        expect(lines.map(line => [line.what, line.said])).toEqual([
            ["Removed a GIF", {gifUrl: "https://static.klipy.com/a.webp"}],
            ["Removed a message", {text: "you all suck"}],
            ["Unmuted", undefined],
            ["Muted for 2h 30m", undefined],
        ])
    })
})

describe("actorOf", () => {
    it("tells Claude, the operator and the console apart", () => {
        expect(actorOf("account-operator", "Claude")).toEqual({kind: "app", name: "Claude"})
        expect(actorOf("account-operator", "")).toEqual({kind: "operator", accountId: "account-operator"})
        expect(actorOf("", "")).toEqual({kind: "console"})
    })
})
