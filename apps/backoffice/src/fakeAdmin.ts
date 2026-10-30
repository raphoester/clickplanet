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
    GetRulesRequest as AntibotGetRulesRequest,
    GetRulesResponse as AntibotGetRulesResponse,
    InspectPlayerRequest,
    InspectPlayerResponse,
    RuleSet as AntibotRuleSet,
    SaveRuleSetRequest as AntibotSaveRuleSetRequest,
    SaveRuleSetResponse as AntibotSaveRuleSetResponse,
    UnbanPlayerRequest,
    UnbanPlayerResponse,
} from "./gen/grpc/antibot/v1/admin_pb.ts"
import {Rules} from "./gen/grpc/antibot/v1/rules_pb.ts"
import {Rules as PlanetRules} from "./gen/grpc/planet/v1/rules_pb.ts"
import {
    FindPlayersRequest,
    FindPlayersResponse,
    GetReplayRequest,
    GetReplayResponse,
    GetRulesRequest,
    GetRulesResponse,
    GrantChargesRequest,
    GrantChargesResponse,
    PaintRandomTilesRequest,
    PaintRandomTilesResponse,
    Player,
    ReassignCountryRequest,
    ReassignCountryResponse,
    RevertPlayerRequest,
    RevertPlayerResponse,
    RuleSet,
    SaveRuleSetRequest,
    SaveRuleSetResponse,
    TopPlayersRequest,
    TopPlayersResponse,
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
import {GetTerritoriesRequest, GetTerritoriesResponse, Territory} from "./gen/grpc/planet/v1/internal_pb.ts"
import {GetLedgerRequest, GetLedgerResponse, LedgerChange} from "./gen/grpc/planet/v1/ledger_pb.ts"
import {
    AddEntryRequest,
    AddEntryResponse,
    AddSeasonRequest,
    AddSeasonResponse,
    DeleteEntryRequest,
    DeleteEntryResponse,
    DeleteSeasonRequest,
    DeleteSeasonResponse,
    EditEntryRequest,
    EditEntryResponse,
    EditSeasonRequest,
    EditSeasonResponse,
    Entry,
    GetScheduleRequest,
    GetScheduleResponse,
    GetSeasonsRequest,
    GetSeasonsResponse,
    Playing,
    ScheduledFinale,
    ScheduledSeason,
    SeasonFinale,
} from "./gen/grpc/gamemaster/v1/admin_pb.ts"
import {Day, EntryKind, Season} from "./gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {ChargesHeld, TileUpdate} from "./gen/grpc/planet/v1/planet_pb.ts"
import {
    AdminGetPlayerRequest,
    AdminGetPlayerResponse,
    PlayerMatch,
    ReconcileTitlesRequest,
    ReconcileTitlesResponse,
    SearchPlayersRequest,
    SearchPlayersResponse,
} from "./gen/grpc/player/v1/admin_pb.ts"
import {Author, GetAuthorsRequest, GetAuthorsResponse} from "./gen/grpc/player/v1/internal_pb.ts"
import {CountryTiles, GetRosterRequest, GetRosterResponse, RosterEntry, Stats} from "./gen/grpc/player/v1/player_pb.ts"
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
    AntibotAdmin, ChatAdmin, ClipsAdmin, Clients, GamemasterAdmin, PlanetAdmin, PlayerAdmin, SupportAdmin,
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

// What each fake was asked, and a refusal to answer with instead, by method.
export class FakeAdmin implements ChatAdmin, PlanetAdmin, PlayerAdmin, SupportAdmin, ClipsAdmin {
    readonly kind = "fake admin"
    readonly antibot: FakeAntibot = new FakeAntibot(this)
    readonly gamemaster: FakeGamemaster = new FakeGamemaster(this)
    readonly asked: {method: string, request: unknown}[] = []
    readonly refusals = new Map<string, Code>()
    private readonly holds = new Map<string, Promise<void>>()
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
    // The set the migration keeps, in part: the fake keeps what it is given and checks nothing.
    antibotRules = new AntibotGetRulesResponse({
        sets: [new AntibotRuleSet({
            name: "default",
            rules: Rules.fromJson({
                bans: {enforce: true, banDurations: ["86400s", "604800s", "94608000s"], reflagInterval: "300s"},
                jury: {minSuspects: 2, suspicionWindow: "600s"},
                retaker: {enabled: true}, sequencer: {enabled: true}, metronome: {enabled: true},
                defender: {enabled: false}, catcher: {enabled: true}, cohort: {enabled: true, minMembers: 2, certainMembers: 6},
                scraper: {enabled: true}, churner: {enabled: true}, hopper: {enabled: true}, rotator: {enabled: true},
            }),
        })],
        playing: "default",
    })
    // A battle that ended, a plain entry that runs until 2098, and a battle to come.
    schedule = new GetScheduleResponse({
        entries: [
            new Entry({
                id: "entry-ended", ruleSet: "rush", kind: EntryKind.BATTLE, description: "Triple points. A bonus box every minute.", times: 3,
                startsAt: Timestamp.fromDate(new Date("2020-10-31T21:00:00Z")),
                endsAt: Timestamp.fromDate(new Date("2020-10-31T23:00:00Z")), priority: 1, madeAt: Timestamp.fromDate(AT),
            }),
            new Entry({
                id: "entry-running", ruleSet: "calm", kind: EntryKind.PLAIN, startsAt: Timestamp.fromDate(AT),
                endsAt: Timestamp.fromDate(new Date("2098-01-01T00:00:00Z")), priority: 1, madeAt: Timestamp.fromDate(AT),
            }),
            new Entry({
                id: "entry-to-come", ruleSet: "frozen", kind: EntryKind.BATTLE, description: "Frozen tiles.", times: 4,
                startsAt: Timestamp.fromDate(new Date("2098-06-01T20:00:00Z")),
                endsAt: Timestamp.fromDate(new Date("2098-06-01T22:00:00Z")), priority: 2, madeAt: Timestamp.fromDate(AT),
            }),
        ],
        playing: new Playing({ruleSet: "calm", kind: EntryKind.PLAIN, since: Timestamp.fromDate(AT), entryId: "entry-running"}),
    })
    // A season that ended, one that runs until 2098 after a day's break, and one of 90 days to come that ends with a
    // final battle.
    seasons = seasonsOf(new Date("2020-09-01T00:00:00Z"), [
        {breakBefore: 0, length: (AT.getTime() - new Date("2020-09-01T00:00:00Z").getTime()) / 1000},
        {breakBefore: DAY, length: (new Date("2098-01-01T00:00:00Z").getTime() - AT.getTime()) / 1000 - DAY},
        {
            breakBefore: 0, length: 90 * DAY,
            finale: new SeasonFinale({length: new Duration({seconds: BigInt(2 * 3600)}), ruleSet: "rush", description: "Triple points.", times: 3}),
        },
    ].map(({breakBefore, length, finale}) => ({
        breakBefore: new Duration({seconds: BigInt(breakBefore)}), length: new Duration({seconds: BigInt(length)}), finale,
    })))
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
    ledgerPages: LedgerChange[][] = []
    territories = new GetTerritoriesResponse({
        tiles: 262119,
        territories: [new Territory({countryId: "fr", tiles: 1200}), new Territory({countryId: "dz", tiles: 800})],
    })
    // The set the migration keeps: the game's own rules.
    rules = new GetRulesResponse({sets: [new RuleSet({name: "default", rules: PlanetRules.fromJson({
        clicks: {
            tempo: 1, perSecond: 0.2, burst: 60, scopeMultiplier: 10, v6ScopeMultiplier: 4, guestScopeMultiplier: 1,
            linkedMultiplier: 2, newAccountClicks: 10,
        },
        toll: {steps: [{share: 0.1, slowdown: 1.5}, {share: 0.2, slowdown: 2.5}, {share: 0.3, slowdown: 4}]},
        bonuses: {
            boxes: {minInterval: "240s", maxInterval: "480s", maxChargesPerHour: 12},
            kinds: {refill: 5, spreadClicks: 3, encloseClicks: 2, bomb: 1, shields: 3},
            offerTtl: "15s", missRetry: "120s", activeWithin: "120s",
            spread: {clicks: 8, maxPerBox: 4},
            bomb: {rings: 4},
            enclose: {maxTiles: 25, held: 3, maxPerBox: 3},
            shield: {held: 30, minPerBox: 5, maxPerBox: 20, perTile: 10},
            welcome: {within: "600s", boxAfter: "60s"},
            quiz: {
                enabled: true, minInterval: "240s", maxInterval: "480s", missRetry: "120s", offerTtl: "25s",
                answerWindow: "8s", leaderBias: 1, maxChargesPerHour: 12,
            },
            signInRefill: true,
        },
    })})], playing: "default"})
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
        return {
            antibot: this.antibot, chat: this, planet: this, player: this, gamemaster: this.gamemaster, support: this, clips: this,
        }
    }

    refuse(method: string, code: Code) {
        this.refusals.set(method, code)
    }

    askedFor(method: string): unknown[] {
        return this.asked.filter(call => call.method === method).map(call => call.request)
    }

    // The calls of a method answer only once the function it gives back is called.
    hold(method: string): () => void {
        let release = () => {}
        this.holds.set(method, new Promise<void>(resolve => release = resolve))
        return () => {
            this.holds.delete(method)
            release()
        }
    }

    answer<T>(method: string, request: unknown, value: T): Promise<T> {
        this.asked.push({method, request})
        const code = this.refusals.get(method)
        const settle = () => code === undefined ? Promise.resolve(value) : Promise.reject(new ConnectError("refused", code))
        const held = this.holds.get(method)
        return held === undefined ? settle() : held.then(settle)
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

    revertPlayer(request: PartialMessage<RevertPlayerRequest>): Promise<RevertPlayerResponse> {
        const restored = request.dryRun === true ? 0 : 118
        return this.answer("revertPlayer", request, new RevertPlayerResponse({touched: 900, held: 120, restored}))
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

    // The ledger's first page, then each of ledgerPages.
    getLedger(request: PartialMessage<GetLedgerRequest>): Promise<GetLedgerResponse> {
        const pages = [this.ledger.changes, ...this.ledgerPages]
        const index = request.page ? Number(request.page) : 0
        const next = index + 1 < pages.length ? String(index + 1) : ""
        return this.answer("getLedger", request, new GetLedgerResponse({changes: pages[index] ?? [], nextPage: next}))
    }

    getTerritories(request: PartialMessage<GetTerritoriesRequest>): Promise<GetTerritoriesResponse> {
        return this.answer("getTerritories", request, this.territories)
    }

    getRules(request: PartialMessage<GetRulesRequest>): Promise<GetRulesResponse> {
        return this.answer("getRules", request, this.rules)
    }

    // Kept as the server keeps it: a set of the same name is replaced.
    saveRuleSet(request: PartialMessage<SaveRuleSetRequest>): Promise<SaveRuleSetResponse> {
        const answer = this.answer("saveRuleSet", request, new SaveRuleSetResponse())
        if (!this.refusals.has("saveRuleSet") && request.set !== undefined) {
            const saved = new RuleSet(request.set)
            const others = this.rules.sets.filter(set => set.name !== saved.name)
            this.rules = new GetRulesResponse({
                sets: [...others, saved].sort((a, b) => Number(a.name !== "default") - Number(b.name !== "default") ||
                    a.name.localeCompare(b.name)),
                playing: this.rules.playing,
            })
        }
        return answer
    }

    getRoster(request: PartialMessage<GetRosterRequest>): Promise<GetRosterResponse> {
        return this.answer("getRoster", request, new GetRosterResponse({entries: this.roster}))
    }

    reconcileTitles(request: PartialMessage<ReconcileTitlesRequest>): Promise<ReconcileTitlesResponse> {
        return this.answer("reconcileTitles", request, new ReconcileTitlesResponse({granted: 12, revoked: 3}))
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

// antibot.v1's calls, asked of the same fake: planet.v1 has a getRules and a saveRuleSet of its own.
export class FakeAntibot implements AntibotAdmin {
    constructor(private readonly admin: FakeAdmin) {}

    banPlayer(request: PartialMessage<BanPlayerRequest>): Promise<BanPlayerResponse> {
        return this.admin.answer("banPlayer", request,
            new BanPlayerResponse({bannedUntil: Timestamp.fromDate(AT), offence: 2, enforced: true}))
    }

    unbanPlayer(request: PartialMessage<UnbanPlayerRequest>): Promise<UnbanPlayerResponse> {
        return this.admin.answer("unbanPlayer", request,
            new UnbanPlayerResponse({offence: 2, bannedUntil: Timestamp.fromDate(AT), pardonedUntil: Timestamp.fromDate(AT)}))
    }

    inspectPlayer(request: PartialMessage<InspectPlayerRequest>): Promise<InspectPlayerResponse> {
        return this.admin.answer("inspectPlayer", request, this.admin.inspection)
    }

    getRules(request: PartialMessage<AntibotGetRulesRequest>): Promise<AntibotGetRulesResponse> {
        return this.admin.answer("getRules", request, this.admin.antibotRules)
    }

    // Kept as the server keeps it: a set of the same name is replaced.
    saveRuleSet(request: PartialMessage<AntibotSaveRuleSetRequest>): Promise<AntibotSaveRuleSetResponse> {
        const answer = this.admin.answer("saveRuleSet", request, new AntibotSaveRuleSetResponse())
        if (!this.admin.refusals.has("saveRuleSet") && request.set !== undefined) {
            const saved = new AntibotRuleSet(request.set)
            const others = this.admin.antibotRules.sets.filter(set => set.name !== saved.name)
            this.admin.antibotRules = new AntibotGetRulesResponse({
                sets: [...others, saved].sort((a, b) => Number(a.name !== "default") - Number(b.name !== "default") ||
                    a.name.localeCompare(b.name)),
                playing: this.admin.antibotRules.playing,
            })
        }
        return answer
    }
}

// Keeps the schedule's entries and the seasons as the server does, and checks nothing.
export class FakeGamemaster implements GamemasterAdmin {
    constructor(private readonly admin: FakeAdmin) {}

    getSchedule(request: PartialMessage<GetScheduleRequest>): Promise<GetScheduleResponse> {
        const finales = this.admin.seasons.seasons.filter(scheduled => scheduled.finale !== undefined).map(scheduled => {
            const endsAt = scheduled.season?.endsAt?.toDate() ?? new Date()
            const startsAt = new Date(endsAt.getTime() - Number(scheduled.finale?.length?.seconds ?? 0n) * 1000)
            return new ScheduledFinale({
                season: scheduled.season?.number, finale: scheduled.finale, startsAt: Timestamp.fromDate(startsAt),
                endsAt: Timestamp.fromDate(endsAt),
            })
        })
        return this.admin.answer("getSchedule", request, new GetScheduleResponse({...this.admin.schedule, finales}))
    }

    addEntry(request: PartialMessage<AddEntryRequest>): Promise<AddEntryResponse> {
        const id = `entry-${this.admin.schedule.entries.length + 1}`
        const answer = this.admin.answer("addEntry", request, new AddEntryResponse({id}))
        if (!this.admin.refusals.has("addEntry")) {
            this.keep([...this.admin.schedule.entries, new Entry({...request, id})])
        }
        return answer
    }

    editEntry(request: PartialMessage<EditEntryRequest>): Promise<EditEntryResponse> {
        const answer = this.admin.answer("editEntry", request, new EditEntryResponse())
        if (!this.admin.refusals.has("editEntry")) {
            this.keep(this.admin.schedule.entries.map(entry => entry.id === request.id ? new Entry({...entry, ...request}) : entry))
        }
        return answer
    }

    deleteEntry(request: PartialMessage<DeleteEntryRequest>): Promise<DeleteEntryResponse> {
        const answer = this.admin.answer("deleteEntry", request, new DeleteEntryResponse())
        if (!this.admin.refusals.has("deleteEntry")) {
            this.keep(this.admin.schedule.entries.filter(entry => entry.id !== request.id))
        }
        return answer
    }

    getSeasons(request: PartialMessage<GetSeasonsRequest>): Promise<GetSeasonsResponse> {
        return this.admin.answer("getSeasons", request, this.admin.seasons)
    }

    addSeason(request: PartialMessage<AddSeasonRequest>): Promise<AddSeasonResponse> {
        const number = this.admin.seasons.seasons.length
        const answer = this.admin.answer("addSeason", request, new AddSeasonResponse({number}))
        if (!this.admin.refusals.has("addSeason")) {
            this.keepSeasons([...this.plans(), planOf(request)])
        }
        return answer
    }

    editSeason(request: PartialMessage<EditSeasonRequest>): Promise<EditSeasonResponse> {
        const answer = this.admin.answer("editSeason", request, new EditSeasonResponse())
        if (!this.admin.refusals.has("editSeason")) {
            this.keepSeasons(this.plans().map((plan, number) => number === request.number ? planOf(request) : plan))
        }
        return answer
    }

    deleteSeason(request: PartialMessage<DeleteSeasonRequest>): Promise<DeleteSeasonResponse> {
        const answer = this.admin.answer("deleteSeason", request, new DeleteSeasonResponse())
        if (!this.admin.refusals.has("deleteSeason")) {
            this.keepSeasons(this.plans().filter((_, number) => number !== request.number))
        }
        return answer
    }

    private plans(): Plan[] {
        return this.admin.seasons.seasons.map(scheduled => ({
            breakBefore: scheduled.breakBefore ?? new Duration(), length: scheduled.length ?? new Duration(), finale: scheduled.finale,
        }))
    }

    private keepSeasons(plans: Plan[]) {
        const first = this.admin.seasons.seasons[0]?.season?.startsAt?.toDate() ?? new Date()
        this.admin.seasons = seasonsOf(first, plans)
    }

    private keep(entries: Entry[]) {
        this.admin.schedule = new GetScheduleResponse({...this.admin.schedule, entries})
    }
}

type Plan = {breakBefore: Duration, length: Duration, finale?: SeasonFinale}

function planOf(request: PartialMessage<AddSeasonRequest> | PartialMessage<EditSeasonRequest>): Plan {
    return {
        breakBefore: new Duration(request.breakBefore), length: new Duration(request.length),
        finale: request.finale === undefined ? undefined : new SeasonFinale(request.finale),
    }
}

const DAY = 24 * 3600

// Season 0 starts at its own start; each next one its break after the end before, as the server works them out.
function seasonsOf(first: Date, plans: Plan[]): GetSeasonsResponse {
    let end: Date | undefined
    return new GetSeasonsResponse({
        seasons: plans.map((plan, number) => {
            const startsAt = end === undefined ? first : new Date(end.getTime() + Number(plan.breakBefore.seconds) * 1000)
            end = new Date(startsAt.getTime() + Number(plan.length.seconds) * 1000)
            return new ScheduledSeason({
                season: new Season({number, startsAt: Timestamp.fromDate(startsAt), endsAt: Timestamp.fromDate(end)}),
                breakBefore: plan.breakBefore, length: plan.length, finale: plan.finale, days: daysOf(startsAt, end),
            })
        }),
    })
}

// A season's days at a 21:00 UTC cutoff, as the server works them out; none for a season that runs for years.
function daysOf(startsAt: Date, endsAt: Date): Day[] {
    if (endsAt.getTime() - startsAt.getTime() > 120 * DAY * 1000) return []
    const days: Day[] = []
    let start = startsAt
    let end = new Date(Date.UTC(start.getUTCFullYear(), start.getUTCMonth(), start.getUTCDate(), 21))
    if (end <= start) end = new Date(end.getTime() + DAY * 1000)
    while (end <= endsAt) {
        days.push(new Day({number: days.length + 1, startsAt: Timestamp.fromDate(start), endsAt: Timestamp.fromDate(end)}))
        start = end
        end = new Date(end.getTime() + DAY * 1000)
    }
    return days
}
