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
    Conviction,
    FindPlayersRequest,
    FindPlayersResponse,
    GetLedgerRequest,
    GetLedgerResponse,
    GetReplayRequest,
    LedgerChange,
    GetReplayResponse,
    GrantChargesRequest,
    GrantChargesResponse,
    InspectPlayerRequest,
    InspectPlayerResponse,
    ListConvictionsRequest,
    ListConvictionsResponse,
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
    WatchdogReading,
} from "./gen/grpc/planet/v1/admin_pb.ts"
import {
    ListRowsRequest,
    ListRowsResponse,
    ListSnapshotsRequest,
    ListSnapshotsResponse,
    StatsRow,
    StatsSnapshot,
    TikTokAccount,
} from "./gen/grpc/clips/v1/admin_pb.ts"
import {
    Bucket,
    GetMetricsRequest,
    GetMetricsResponse,
    Histogram,
    Label,
    Metric,
} from "./gen/grpc/metrics/v1/admin_pb.ts"
import {GetTerritoriesRequest, GetTerritoriesResponse, Territory} from "./gen/grpc/planet/v1/internal_pb.ts"
import {ChargesHeld, TileUpdate} from "./gen/grpc/planet/v1/planet_pb.ts"
import {
    AdminGetPlayerRequest,
    AdminGetPlayerResponse,
    PlayerMatch,
    ReconcileTitlesRequest,
    ReconcileTitlesResponse,
    RecountStatsRequest,
    RecountStatsResponse,
    SearchPlayersRequest,
    SearchPlayersResponse,
} from "./gen/grpc/player/v1/admin_pb.ts"
import {Author, GetAuthorsRequest, GetAuthorsResponse} from "./gen/grpc/player/v1/internal_pb.ts"
import {CountryTiles, GetRosterRequest, GetRosterResponse, RosterEntry, Stats} from "./gen/grpc/player/v1/player_pb.ts"
import {GetRaceRequest, GetRaceResponse, Race, Round, RoundStanding, Score} from "./gen/grpc/seasons/v1/seasons_pb.ts"
import {
    DecideReportRequest,
    DecideReportResponse,
    ListNotesRequest,
    ListNotesResponse,
    ListReportsRequest,
    ListReportsResponse,
    Note,
    NoteKind,
    Report,
} from "./gen/grpc/support/v1/admin_pb.ts"
import {
    ChatAdmin,
    ClipsAdmin,
    Clients,
    MetricsAdmin,
    PlanetAdmin,
    PlayerAdmin,
    SeasonsAdmin,
    SupportAdmin,
} from "./clients.ts"

export const AT = new Date("2026-10-18T12:00:00Z")

export const ACCOUNT = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

// A change of the fake ledger, its minutes into the hour before AT.
function change(minutes: number, by: string, update: PartialMessage<TileUpdate>): LedgerChange {
    return new LedgerChange({
        at: Timestamp.fromDate(new Date(AT.getTime() - 3600_000 + minutes * 60_000)),
        update: new TileUpdate(update), by,
    })
}

function at(seconds: number): Timestamp {
    return Timestamp.fromDate(new Date(AT.getTime() + seconds * 1000))
}

function counter(name: string, value: number, labels: Record<string, string> = {}): Metric {
    return new Metric({
        name, value: {case: "counter", value},
        labels: Object.entries(labels).map(([key, label]) => new Label({name: key, value: label})),
    })
}

function gauge(name: string, value: number, labels: Record<string, string> = {}): Metric {
    return new Metric({
        name, value: {case: "gauge", value},
        labels: Object.entries(labels).map(([key, label]) => new Label({name: key, value: label})),
    })
}

// /metrics as the server reads it `seconds` after AT, `minutes` minutes of play after the fake process started.
export function metricsAt(seconds: number, minutes: number): GetMetricsResponse {
    return new GetMetricsResponse({
        startedAt: at(-3600), readAt: at(seconds),
        metrics: [
            counter("clicks_total", 300 * minutes, {country_id: "fr", status: "ok"}),
            counter("clicks_total", 100 * minutes, {country_id: "dz", status: "ok"}),
            counter("throttled_clicks", 30 * minutes),
            counter("banned_clicks", 3 * minutes),
            counter("blocked_clicks", 0),
            gauge("banned_callers", 2),
            gauge("antibot_opinions_standing", 4, {watchdog: "metronome", level: "suspect"}),
            gauge("antibot_opinions_standing", 1, {watchdog: "metronome", level: "certain"}),
            gauge("antibot_opinions_standing", 1, {watchdog: "catcher", level: "suspect"}),
            gauge("antibot_opinions_standing", 0, {watchdog: "catcher", level: "certain"}),
            counter("ban_flags", minutes, {watchdog: "catcher"}),
            counter("bonus_offers_total", 20 * minutes),
            counter("bonus_claims_total", 12 * minutes, {outcome: "granted"}),
            counter("bonus_claims_total", minutes, {outcome: "refused"}),
            counter("bonus_lapsed_total", 8 * minutes),
            counter("bonus_claims_foreign_total", minutes),
            new Metric({
                name: "bonus_catch_seconds",
                value: {
                    case: "histogram", value: new Histogram({
                        count: BigInt(12 * minutes), sum: 18 * minutes,
                        buckets: [
                            new Bucket({upperBound: 0.5, count: BigInt(minutes)}),
                            new Bucket({upperBound: 1, count: BigInt(3 * minutes)}),
                            new Bucket({upperBound: 3, count: BigInt(12 * minutes)}),
                        ],
                    }),
                },
            }),
            counter("process_cpu_seconds_total", 30 * minutes),
            gauge("process_resident_memory_bytes", 300 * 1024 * 1024),
            gauge("go_goroutines", 420),
            counter("events_dropped_total", 0, {event: "planet.v1.TileTaken", subscriber: "seasons-standings"}),
        ],
    })
}

// What each fake was asked, and a refusal to answer with instead, by method.
export class FakeAdmin implements ChatAdmin, PlanetAdmin, PlayerAdmin, SupportAdmin, ClipsAdmin, MetricsAdmin, SeasonsAdmin {
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
    // A bug from a named account, and an idea whose sender deleted its account.
    notes = [
        new Note({
            id: "note-bug", kind: NoteKind.BUG, accountId: ACCOUNT, text: "the globe goes black",
            submittedAt: Timestamp.fromDate(AT),
        }),
        new Note({
            id: "note-idea", kind: NoteKind.SUGGESTION, accountId: "", text: "more flags",
            submittedAt: Timestamp.fromDate(new Date(AT.getTime() - 60_000)),
        }),
    ]

    reports = [new Report({id: "report-1", reporterAccountId: "account-reporter", reportedAccountId: "account-reported"})]
    // Four clicks of BraveFox42 (three taken, one struck), and a tile a guest's spread took back from France.
    ledger = new GetLedgerResponse({
        changes: [
            change(1, `account:${ACCOUNT}`, {tileId: 1, countryId: "fr", clicked: true}),
            change(2, `account:${ACCOUNT}`, {tileId: 2, countryId: "fr", previousCountryId: "dz", clicked: true}),
            change(3, "scope:203.0.113.7", {tileId: 2, countryId: "dz", previousCountryId: "fr"}),
            change(4, `account:${ACCOUNT}`, {tileId: 3, countryId: "dz", previousCountryId: "dz", clicked: true, shields: 2}),
            change(5, `account:${ACCOUNT}`, {tileId: 4, countryId: "fr", previousCountryId: "dz", clicked: true}),
        ],
    })
    territories = new GetTerritoriesResponse({
        tiles: 262119,
        territories: [new Territory({countryId: "fr", tiles: 1200}), new Territory({countryId: "dz", tiles: 800})],
    })
    // BraveFox42 caught every box, the guest it rotates with fell beside it, and an address with no account walked the ids.
    convictions = [
        new Conviction({
            at: at(-600), scope: "203.0.113.7", accountId: ACCOUNT, flags: 1, offence: 1, bannedUntil: at(86_400),
            readings: [
                new WatchdogReading({watchdog: "retaker", level: "clear"}),
                new WatchdogReading({watchdog: "catcher", level: "certain", evidence: "catch median=400ms", at: at(-610)}),
                new WatchdogReading({watchdog: "metronome", level: "suspect", evidence: "stamina busy=5h40m", at: at(-620)}),
            ],
            clicks: 900, activeFor: new Duration({seconds: 1800n}), longestGap: new Duration({seconds: 2n}),
            topCountry: "fr", topCountryClicks: 880,
        }),
        new Conviction({
            at: at(-600), scope: "203.0.113.7", accountId: "account-guest", alongsideAccountId: ACCOUNT, flags: 1,
            offence: 1, bannedUntil: at(86_400),
            readings: [new WatchdogReading({watchdog: "catcher", level: "certain", evidence: "catch median=400ms", at: at(-610)})],
            clicks: 900, topCountry: "fr", topCountryClicks: 880,
        }),
        new Conviction({
            at: at(-60), scope: "2001:db8:1:2::/64", flags: 1, offence: 2, bannedUntil: at(604_800),
            readings: [new WatchdogReading({watchdog: "sequencer", level: "certain", evidence: "walk steps=200", at: at(-61)})],
            clicks: 240, topCountry: "dz", topCountryClicks: 240,
        }),
    ]
    // GetMetrics answers each in turn, then the last again.
    metricReads = [metricsAt(-60, 59), metricsAt(0, 60)]
    race = new Race({
        round: new Round({
            number: 12, endsAtUnixMs: BigInt(AT.getTime() + 3600_000), finale: true,
            standings: [
                new RoundStanding({rank: 1, countryId: "fr", share: 0.31, points: 25}),
                new RoundStanding({rank: 2, countryId: "dz", share: 0.22, points: 18}),
            ],
        }),
        scores: [
            new Score({rank: 1, countryId: "dz", points: 210, roundsWon: 6}),
            new Score({rank: 2, countryId: "fr", points: 190, roundsWon: 4}),
        ],
    })
    roster = [
        new RosterEntry({key: "1", accountId: ACCOUNT, name: "BraveFox42", countryId: "fr"}),
        new RosterEntry({key: "2", accountId: "account-guest", name: "guest_a1b2c3", countryId: "dz", guest: true}),
    ]

    // Yesterday's snapshot, TikTok still personal; and the one of the day before.
    snapshots = [
        new StatsSnapshot({
            takenAt: Timestamp.fromDate(AT), posts: 40, notOut: 3, rows: 73, syncing: 4, curves: 30,
            tiktokAccounts: [new TikTokAccount({
                accountId: "tt-1", username: "clickplanet.lol", accountType: "personal", rows: 37,
                commercialMusicRefusal: "zernio answered 400 to the commercial music library: Not a business account",
            })],
        }),
        new StatsSnapshot({takenAt: Timestamp.fromDate(new Date(AT.getTime() - 86_400_000)), posts: 38, rows: 70}),
    ]
    // ListRows answers them a page at a time, the page token is the index of the page.
    rowPages: StatsRow[][] = [[
        new StatsRow({
            takenAt: Timestamp.fromDate(AT), postId: "post-1", platform: "tiktok",
            document: JSON.stringify({
                takenAt: AT.toISOString(), postId: "post-1", platform: "tiktok", ageHours: 12.5,
                answer: {httpStatus: 200}, analytics: {analytics: {views: 812}},
                metadata: {storyFlag: "fr", record: {clip: {variants: {length: "short"}}}},
            }),
        }),
    ]]

    clients(): Clients {
        return {chat: this, planet: this, player: this, support: this, clips: this, metrics: this, seasons: this}
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

    listSnapshots(request: PartialMessage<ListSnapshotsRequest>): Promise<ListSnapshotsResponse> {
        return this.answer("listSnapshots", request, new ListSnapshotsResponse({snapshots: this.snapshots}))
    }

    listRows(request: PartialMessage<ListRowsRequest>): Promise<ListRowsResponse> {
        const index = request.page ? Number(request.page) : 0
        const next = index + 1 < this.rowPages.length ? String(index + 1) : ""
        return this.answer("listRows", request, new ListRowsResponse({rows: this.rowPages[index] ?? [], nextPage: next}))
    }

    getMessage(request: PartialMessage<GetMessageRequest>): Promise<GetMessageResponse> {
        return this.answer("getMessage", request, this.message)
    }

    // A page of chat as the server reads it: oldest first, just before or just after the cursor, the newest with none,
    // and one account's messages alone when it names one. The cursor may be anybody's message.
    listMessages(request: PartialMessage<ListMessagesRequest>): Promise<ListMessagesResponse> {
        const limit = request.limit || 50
        const cursor = request.beforeMessageId || request.afterMessageId || ""
        const at = this.chat.findIndex(message => message.messageId === cursor)
        if (cursor !== "" && at < 0) {
            this.asked.push({method: "listMessages", request})
            return Promise.reject(new ConnectError("no message kept under this id", Code.NotFound))
        }
        const kept = this.chat
            .map((message, index) => ({message, index}))
            .filter(({message}) => !request.accountId || message.accountId === request.accountId)
        const page = request.afterMessageId
            ? kept.filter(({index}) => index > at).slice(0, limit)
            : kept.filter(({index}) => cursor === "" || index < at).slice(-limit)
        return this.answer("listMessages", request, new ListMessagesResponse({messages: page.map(({message}) => message)}))
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
            new UnbanPlayerResponse({offence: 2, bannedUntil: Timestamp.fromDate(AT), pardonedUntil: Timestamp.fromDate(AT)}))
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

    getLedger(request: PartialMessage<GetLedgerRequest>): Promise<GetLedgerResponse> {
        return this.answer("getLedger", request, this.ledger)
    }

    listConvictions(request: PartialMessage<ListConvictionsRequest>): Promise<ListConvictionsResponse> {
        return this.answer("listConvictions", request, new ListConvictionsResponse({convictions: this.convictions}))
    }

    getMetrics(request: PartialMessage<GetMetricsRequest>): Promise<GetMetricsResponse> {
        const read = this.metricReads.length > 1 ? this.metricReads.shift() : this.metricReads[0]
        return this.answer("getMetrics", request, read ?? new GetMetricsResponse())
    }

    getRace(request: PartialMessage<GetRaceRequest>): Promise<GetRaceResponse> {
        return this.answer("getRace", request, new GetRaceResponse({race: this.race}))
    }

    getTerritories(request: PartialMessage<GetTerritoriesRequest>): Promise<GetTerritoriesResponse> {
        return this.answer("getTerritories", request, this.territories)
    }

    getRoster(request: PartialMessage<GetRosterRequest>): Promise<GetRosterResponse> {
        return this.answer("getRoster", request, new GetRosterResponse({entries: this.roster}))
    }

    reconcileTitles(request: PartialMessage<ReconcileTitlesRequest>): Promise<ReconcileTitlesResponse> {
        return this.answer("reconcileTitles", request, new ReconcileTitlesResponse({granted: 12, revoked: 3}))
    }

    recountStats(request: PartialMessage<RecountStatsRequest>): Promise<RecountStatsResponse> {
        return this.answer("recountStats", request, new RecountStatsResponse({recounted: 164}))
    }

    getAuthors(request: PartialMessage<GetAuthorsRequest>): Promise<GetAuthorsResponse> {
        const asked = new Set(request.accountIds)
        return this.answer("getAuthors", request,
            new GetAuthorsResponse({authors: this.authors.filter(author => asked.has(author.accountId))}))
    }

    // The accounts the fake names, matched as the server does: in any case, anywhere in the name.
    searchPlayers(request: PartialMessage<SearchPlayersRequest>): Promise<SearchPlayersResponse> {
        const name = (request.name ?? "").trim().toLowerCase()
        const players = this.authors
            .filter(author => author.name.toLowerCase().includes(name))
            .map(author => new PlayerMatch({accountId: author.accountId, name: author.name, guest: author.guest}))
        return this.answer("searchPlayers", request, new SearchPlayersResponse({players, total: players.length}))
    }

    getPlayer(request: PartialMessage<AdminGetPlayerRequest>): Promise<AdminGetPlayerResponse> {
        return this.answer("getPlayer", request, this.profile)
    }

    listNotes(request: PartialMessage<ListNotesRequest>): Promise<ListNotesResponse> {
        const kind = request.kind ?? NoteKind.UNSPECIFIED
        const notes = this.notes.filter(note => kind === NoteKind.UNSPECIFIED || note.kind === kind)
        return this.answer("listNotes", request, new ListNotesResponse({notes}))
    }

    listReports(request: PartialMessage<ListReportsRequest>): Promise<ListReportsResponse> {
        return this.answer("listReports", request, new ListReportsResponse({reports: this.reports}))
    }

    decideReport(request: PartialMessage<DecideReportRequest>): Promise<DecideReportResponse> {
        return this.answer("decideReport", request, new DecideReportResponse({decidedAt: Timestamp.fromDate(AT)}))
    }
}
