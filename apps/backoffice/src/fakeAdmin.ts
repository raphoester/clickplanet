import {Duration, PartialMessage, Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    GetMessageRequest,
    GetMessageResponse,
    KeptMessage,
    KeptQuote,
    MuteRequest,
    MuteResponse,
    RemoveMessageRequest,
    RemoveMessageResponse,
} from "./gen/grpc/chat/v1/admin_pb.ts"
import {
    BanPlayerRequest,
    BanPlayerResponse,
    FindPlayersRequest,
    FindPlayersResponse,
    GetReplayRequest,
    GetReplayResponse,
    GrantChargesRequest,
    GrantChargesResponse,
    InspectPlayerRequest,
    InspectPlayerResponse,
    PaintRandomTilesRequest,
    PaintRandomTilesResponse,
    Player,
    ReassignCountryRequest,
    ReassignCountryResponse,
    RevertPlayerRequest,
    RevertPlayerResponse,
    TopPlayersRequest,
    TopPlayersResponse,
    UnbanPlayerRequest,
    UnbanPlayerResponse,
} from "./gen/grpc/planet/v1/admin_pb.ts"
import {ChargesHeld} from "./gen/grpc/planet/v1/planet_pb.ts"
import {
    NameAccountsRequest,
    NameAccountsResponse,
    ReconcileTitlesRequest,
    ReconcileTitlesResponse,
} from "./gen/grpc/player/v1/admin_pb.ts"
import {DecideReportRequest, DecideReportResponse} from "./gen/grpc/support/v1/admin_pb.ts"
import {ChatAdmin, Clients, PlanetAdmin, PlayerAdmin, SupportAdmin} from "./clients.ts"

export const AT = new Date("2026-10-18T12:00:00Z")

export const ACCOUNT = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

// What each fake was asked, and a refusal to answer with instead, by method.
export class FakeAdmin implements ChatAdmin, PlanetAdmin, PlayerAdmin, SupportAdmin {
    readonly kind = "fake admin"
    readonly asked: {method: string, request: unknown}[] = []
    readonly refusals = new Map<string, Code>()
    // Bob baits, Cleo calms, Ana answers Bob, then Bob again.
    message = new GetMessageResponse({
        messageId: "message-1", accountId: "account-reported", authorName: "Ana", text: "you all suck",
        sentAt: Timestamp.fromDate(AT), countryId: "fr",
        replyTo: new KeptQuote({messageId: "message-before-1", authorName: "Bob", text: "come at me"}),
        before: [
            new KeptMessage({
                messageId: "message-before-1", accountId: "account-reporter", authorName: "Bob", text: "come at me",
                sentAt: Timestamp.fromDate(new Date(AT.getTime() - 120_000)), countryId: "de",
            }),
            new KeptMessage({
                messageId: "message-before-2", accountId: "account-cleo", authorName: "Cleo", text: "calm down",
                sentAt: Timestamp.fromDate(new Date(AT.getTime() - 60_000)), countryId: "it",
            }),
        ],
        after: [
            new KeptMessage({
                messageId: "message-after-1", accountId: "account-reporter", authorName: "Bob", text: "so rude",
                sentAt: Timestamp.fromDate(new Date(AT.getTime() + 60_000)), countryId: "de",
            }),
        ],
    })
    players = [
        new Player({
            accountId: ACCOUNT, scope: "203.0.113.7", tiles: 3, takes: 900, takesPerMinute: 30,
            tilesPerMinute: 0.1, firstAt: Timestamp.fromDate(AT), lastAt: Timestamp.fromDate(AT),
            activeFor: new Duration({seconds: 1800n}),
        }),
        new Player({scope: "2001:db8:1:2::/64", tiles: 40, takes: 50}),
    ]
    inspection = new InspectPlayerResponse({accountId: "account-reported", clicks: 42, banned: false})

    clients(): Clients {
        return {chat: this, planet: this, player: this, support: this}
    }

    refuse(method: string, code: Code) {
        this.refusals.set(method, code)
    }

    askedFor(method: string): unknown[] {
        return this.asked.filter(call => call.method === method).map(call => call.request)
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

    unbanPlayer(request: PartialMessage<UnbanPlayerRequest>): Promise<UnbanPlayerResponse> {
        return this.answer("unbanPlayer", request,
            new UnbanPlayerResponse({offence: 2, bannedUntil: Timestamp.fromDate(AT)}))
    }

    revertPlayer(request: PartialMessage<RevertPlayerRequest>): Promise<RevertPlayerResponse> {
        const restored = request.dryRun === true ? 0 : 118
        return this.answer("revertPlayer", request, new RevertPlayerResponse({touched: 900, held: 120, restored}))
    }

    inspectPlayer(request: PartialMessage<InspectPlayerRequest>): Promise<InspectPlayerResponse> {
        return this.answer("inspectPlayer", request, this.inspection)
    }

    grantCharges(request: PartialMessage<GrantChargesRequest>): Promise<GrantChargesResponse> {
        return this.answer("grantCharges", request, new GrantChargesResponse({
            accountId: request.accountId,
            before: new ChargesHeld({enclosures: 2}),
            after: new ChargesHeld({bomb: true, enclosures: 3}),
        }))
    }

    topPlayers(request: PartialMessage<TopPlayersRequest>): Promise<TopPlayersResponse> {
        return this.answer("topPlayers", request, new TopPlayersResponse({players: this.players, total: 75}))
    }

    findPlayers(request: PartialMessage<FindPlayersRequest>): Promise<FindPlayersResponse> {
        return this.answer("findPlayers", request, new FindPlayersResponse({players: this.players, total: 2}))
    }

    reassignCountry(request: PartialMessage<ReassignCountryRequest>): Promise<ReassignCountryResponse> {
        const answer = request.dryRun === true
            ? {fromBefore: 22040, toBefore: 300, fromAfter: 22040, toAfter: 300}
            : {fromBefore: 22040, toBefore: 300, moved: 22040, fromAfter: 0, toAfter: 22340}
        return this.answer("reassignCountry", request, new ReassignCountryResponse(answer))
    }

    paintRandomTiles(request: PartialMessage<PaintRandomTilesRequest>): Promise<PaintRandomTilesResponse> {
        const painted = request.dryRun === true ? 0 : 498
        return this.answer("paintRandomTiles", request,
            new PaintRandomTilesResponse({eligible: 4000, picked: 500, painted, outsideArea: 12}))
    }

    getReplay(request: PartialMessage<GetReplayRequest>): Promise<GetReplayResponse> {
        return this.answer("getReplay", request, new GetReplayResponse({
            since: request.since ?? Timestamp.fromDate(AT),
            until: request.until ?? Timestamp.fromDate(AT),
        }))
    }

    reconcileTitles(request: PartialMessage<ReconcileTitlesRequest>): Promise<ReconcileTitlesResponse> {
        return this.answer("reconcileTitles", request, new ReconcileTitlesResponse({granted: 12, revoked: 3}))
    }

    nameAccounts(request: PartialMessage<NameAccountsRequest>): Promise<NameAccountsResponse> {
        return this.answer("nameAccounts", request, new NameAccountsResponse({named: 7}))
    }

    decideReport(request: PartialMessage<DecideReportRequest>): Promise<DecideReportResponse> {
        return this.answer("decideReport", request, new DecideReportResponse({decidedAt: Timestamp.fromDate(AT)}))
    }
}
