import {PartialMessage, Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    GetMessageRequest,
    GetMessageResponse,
    MuteRequest,
    MuteResponse,
    RemoveMessageRequest,
    RemoveMessageResponse,
} from "../gen/grpc/chat/v1/admin_pb.ts"
import {
    BanPlayerRequest,
    BanPlayerResponse,
    InspectPlayerRequest,
    InspectPlayerResponse,
} from "../gen/grpc/planet/v1/admin_pb.ts"
import {DecideReportRequest, DecideReportResponse} from "../gen/grpc/support/v1/admin_pb.ts"
import {ChatAdmin, PlanetAdmin, SupportAdmin} from "./actions.ts"
import {Clients} from "./clients.ts"

export const AT = new Date("2026-10-18T12:00:00Z")

// What each fake was asked, and a refusal to answer with instead, by method.
export class FakeAdmin implements ChatAdmin, PlanetAdmin, SupportAdmin {
    readonly kind = "fake admin"
    readonly asked: {method: string, request: unknown}[] = []
    readonly refusals = new Map<string, Code>()
    message = new GetMessageResponse({
        messageId: "message-1", accountId: "account-reported", text: "you all suck",
        sentAt: Timestamp.fromDate(AT), countryId: "fr",
    })

    clients(): Clients {
        return {chat: this, planet: this, support: this}
    }

    refuse(method: string, code: Code) {
        this.refusals.set(method, code)
    }

    private answer<T>(method: string, request: unknown, value: T): Promise<T> {
        this.asked.push({method, request})
        const code = this.refusals.get(method)
        return code === undefined ? Promise.resolve(value) : Promise.reject(new ConnectError("refused", code))
    }

    getMessage(request: PartialMessage<GetMessageRequest>): Promise<GetMessageResponse> {
        return this.answer("getMessage", request, this.message)
    }

    removeMessage(request: PartialMessage<RemoveMessageRequest>): Promise<RemoveMessageResponse> {
        return this.answer("removeMessage", request, new RemoveMessageResponse({removedAt: Timestamp.fromDate(AT)}))
    }

    mute(request: PartialMessage<MuteRequest>): Promise<MuteResponse> {
        return this.answer("mute", request, new MuteResponse({mutedUntil: Timestamp.fromDate(AT)}))
    }

    banPlayer(request: PartialMessage<BanPlayerRequest>): Promise<BanPlayerResponse> {
        return this.answer("banPlayer", request,
            new BanPlayerResponse({bannedUntil: Timestamp.fromDate(AT), offence: 2, enforced: true}))
    }

    inspectPlayer(request: PartialMessage<InspectPlayerRequest>): Promise<InspectPlayerResponse> {
        return this.answer("inspectPlayer", request,
            new InspectPlayerResponse({accountId: "account-reported", clicks: 42, banned: false}))
    }

    decideReport(request: PartialMessage<DecideReportRequest>): Promise<DecideReportResponse> {
        return this.answer("decideReport", request, new DecideReportResponse({decidedAt: Timestamp.fromDate(AT)}))
    }
}
