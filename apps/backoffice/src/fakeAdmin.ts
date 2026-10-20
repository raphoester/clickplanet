import {Duration, PartialMessage, Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    GetMessageRequest,
    GetMessageResponse,
    KeptMessage,
    KeptQuote,
    ListMessagesRequest,
    ListMessagesResponse,
    MuteRequest,
    MuteResponse,
    RemoveMessageRequest,
    RemoveMessageResponse,
} from "./gen/grpc/chat/v1/admin_pb.ts"
import {
    BanPlayerRequest,
    BanPlayerResponse,
    ClearAreaRequest,
    ClearAreaResponse,
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
    RestoreAreaRequest,
    RestoreAreaResponse,
    RevertPlayerRequest,
    RevertPlayerResponse,
    TopPlayersRequest,
    TopPlayersResponse,
    UnbanPlayerRequest,
    UnbanPlayerResponse,
} from "./gen/grpc/planet/v1/admin_pb.ts"
import {ChargesHeld} from "./gen/grpc/planet/v1/planet_pb.ts"
import {
    AdminGetPlayerRequest,
    AdminGetPlayerResponse,
    ReconcileTitlesRequest,
    ReconcileTitlesResponse,
} from "./gen/grpc/player/v1/admin_pb.ts"
import {Author, GetAuthorsRequest, GetAuthorsResponse} from "./gen/grpc/player/v1/internal_pb.ts"
import {CountryTiles, Stats} from "./gen/grpc/player/v1/player_pb.ts"
import {DecideReportRequest, DecideReportResponse} from "./gen/grpc/support/v1/admin_pb.ts"
import {ChatAdmin, Clients, PlanetAdmin, PlayerAdmin, SupportAdmin} from "./clients.ts"

export const AT = new Date("2026-10-18T12:00:00Z")

export const ACCOUNT = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

// What each fake was asked, and a refusal to answer with instead, by method.
export class FakeAdmin implements ChatAdmin, PlanetAdmin, PlayerAdmin, SupportAdmin {
    readonly kind = "fake admin"
    readonly asked: {method: string, request: unknown}[] = []
    readonly refusals = new Map<string, Code>()
    message = new GetMessageResponse({
        messageId: "message-1", accountId: "account-reported", text: "you all suck",
        sentAt: Timestamp.fromDate(AT), countryId: "fr",
    })
    // The chat ListMessages pages through, the reported message among it: Bob baits, Cleo answers him, Ana insults,
    // then Bob again.
    chat = [
        new KeptMessage({
            messageId: "message-before-1", accountId: "account-reporter", authorName: "Bob", text: "come at me",
            sentAt: Timestamp.fromDate(new Date(AT.getTime() - 120_000)), countryId: "de",
        }),
        new KeptMessage({
            messageId: "message-before-2", accountId: "account-cleo", authorName: "Cleo", text: "calm down",
            sentAt: Timestamp.fromDate(new Date(AT.getTime() - 60_000)), countryId: "it",
            replyTo: new KeptQuote({messageId: "message-before-1", authorName: "Bob", text: "come at me"}),
        }),
        new KeptMessage({
            messageId: "message-1", accountId: "account-reported", authorName: "Ana", text: "you all suck",
            sentAt: Timestamp.fromDate(AT), countryId: "fr",
        }),
        new KeptMessage({
            messageId: "message-after-1", accountId: "account-reporter", authorName: "Bob", text: "so rude",
            sentAt: Timestamp.fromDate(new Date(AT.getTime() + 60_000)), countryId: "de",
        }),
    ]
    players = [
        new Player({
            accountId: ACCOUNT, scope: "203.0.113.7", tiles: 3, takes: 900, takesPerMinute: 30,
            tilesPerMinute: 0.1, firstAt: Timestamp.fromDate(AT), lastAt: Timestamp.fromDate(AT),
            activeFor: new Duration({seconds: 1800n}), flagCountryId: "fr",
        }),
        new Player({scope: "2001:db8:1:2::/64", tiles: 40, takes: 50}),
    ]
    inspection = new InspectPlayerResponse({accountId: "account-reported", clicks: 42, banned: false})
    authors = [new Author({accountId: ACCOUNT, name: "BraveFox42"})]
    profile = new AdminGetPlayerResponse({
        name: "BraveFox42", linked: true, createdAtUnixMs: BigInt(AT.getTime()), messagesSent: 4n,
        stats: new Stats({tilesTaken: 1200n, streakCurrent: 3, streakBest: 12}),
        playsFor: [new CountryTiles({countryId: "fr", tiles: 1100n}), new CountryTiles({countryId: "gb-sct", tiles: 100n})],
    })

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

    // A page of chat as the server reads it: oldest first, just before or just after the cursor, the newest with none.
    listMessages(request: PartialMessage<ListMessagesRequest>): Promise<ListMessagesResponse> {
        const limit = request.limit || 50
        const cursor = request.beforeMessageId || request.afterMessageId || ""
        const at = this.chat.findIndex(message => message.messageId === cursor)
        if (cursor !== "" && at < 0) {
            this.asked.push({method: "listMessages", request})
            return Promise.reject(new ConnectError("no message kept under this id", Code.NotFound))
        }
        const messages = request.afterMessageId
            ? this.chat.slice(at + 1, at + 1 + limit)
            : this.chat.slice(Math.max(0, (cursor === "" ? this.chat.length : at) - limit), cursor === "" ? undefined : at)
        return this.answer("listMessages", request, new ListMessagesResponse({messages}))
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

    restoreArea(request: PartialMessage<RestoreAreaRequest>): Promise<RestoreAreaResponse> {
        return this.answer("restoreArea", request, new RestoreAreaResponse({tiles: 37, changed: 12}))
    }

    clearArea(request: PartialMessage<ClearAreaRequest>): Promise<ClearAreaResponse> {
        return this.answer("clearArea", request, new ClearAreaResponse({tiles: 37, owned: 20}))
    }

    reconcileTitles(request: PartialMessage<ReconcileTitlesRequest>): Promise<ReconcileTitlesResponse> {
        return this.answer("reconcileTitles", request, new ReconcileTitlesResponse({granted: 12, revoked: 3}))
    }

    getAuthors(request: PartialMessage<GetAuthorsRequest>): Promise<GetAuthorsResponse> {
        const asked = new Set(request.accountIds)
        return this.answer("getAuthors", request,
            new GetAuthorsResponse({authors: this.authors.filter(author => asked.has(author.accountId))}))
    }

    getPlayer(request: PartialMessage<AdminGetPlayerRequest>): Promise<AdminGetPlayerResponse> {
        return this.answer("getPlayer", request, this.profile)
    }

    decideReport(request: PartialMessage<DecideReportRequest>): Promise<DecideReportResponse> {
        return this.answer("decideReport", request, new DecideReportResponse({decidedAt: Timestamp.fromDate(AT)}))
    }
}
