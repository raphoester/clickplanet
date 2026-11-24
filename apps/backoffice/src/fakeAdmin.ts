import {textBody} from "./chat/said.ts"
import {Duration, PartialMessage, Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    GetMessageRequest,
    GetMessageResponse,
    GetMuteRequest,
    GetMuteResponse,
    KeptMessage,
    KeptQuote,
    ListMessagesRequest,
    ListMessagesResponse,
    ListModerationRequest,
    ListModerationResponse,
    ModerationAction,
    MuteAction,
    MuteRequest,
    MuteResponse,
    RemoveMessageRequest,
    RemoveMessageResponse,
    UnmuteRequest,
    UnmuteResponse,
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
    WarnPlayerRequest,
    WarnPlayerResponse,
} from "./gen/grpc/antibot/v1/admin_pb.ts"
import {Rules} from "./gen/grpc/antibot/v1/rules_pb.ts"
import {
    GetRulesRequest as BonusesGetRulesRequest,
    GetRulesResponse as BonusesGetRulesResponse,
    RuleSet as BonusesRuleSet,
    SaveRuleSetRequest as BonusesSaveRuleSetRequest,
    SaveRuleSetResponse as BonusesSaveRuleSetResponse,
} from "./gen/grpc/bonuses/v1/admin_pb.ts"
import {Rules as BonusesRules} from "./gen/grpc/bonuses/v1/rules_pb.ts"
import {
    GetRulesRequest as BoxesGetRulesRequest,
    GetRulesResponse as BoxesGetRulesResponse,
    RuleSet as BoxesRuleSet,
    SaveRuleSetRequest as BoxesSaveRuleSetRequest,
    SaveRuleSetResponse as BoxesSaveRuleSetResponse,
} from "./gen/grpc/boxes/v1/admin_pb.ts"
import {Rules as BoxesRules} from "./gen/grpc/boxes/v1/rules_pb.ts"
import {
    GetRulesRequest as QuizzesGetRulesRequest,
    GetRulesResponse as QuizzesGetRulesResponse,
    ListQuizQuestionsRequest,
    ListQuizQuestionsResponse,
    RuleSet as QuizzesRuleSet,
    SaveRuleSetRequest as QuizzesSaveRuleSetRequest,
    SaveRuleSetResponse as QuizzesSaveRuleSetResponse,
} from "./gen/grpc/quizzes/v1/admin_pb.ts"
import {Rules as QuizzesRules} from "./gen/grpc/quizzes/v1/rules_pb.ts"
import {MuteKind} from "./gen/grpc/chat/v1/chat_pb.ts"
import {
    DiscordLink,
    GetPlayerLinkRequest,
    GetPlayerLinkResponse,
    LinkMembersRequest,
    LinkMembersResponse,
    UnlinkPlayerRequest,
    UnlinkPlayerResponse,
} from "./gen/grpc/alerts/v1/admin_pb.ts"
import {Rules as PlanetRules} from "./gen/grpc/planet/v1/rules_pb.ts"
import {
    Background as PromoBackground,
    Card,
    CardDraft,
    CreateBackgroundUploadRequest,
    CreateBackgroundUploadResponse,
    GetCardsRequest,
    GetCardsResponse,
    KeepBackgroundRequest,
    KeepBackgroundResponse,
    ListBackgroundsRequest,
    ListBackgroundsResponse,
    ExampleBattle,
    Placeholder,
    RemoveBackgroundRequest,
    RemoveBackgroundResponse,
    SaveCardsRequest,
    SaveCardsResponse,
    SetCards,
} from "./gen/grpc/promo/v1/admin_pb.ts"
import {TeaserCard} from "./gen/grpc/promo/v1/promo_pb.ts"
import {playedBySet} from "./promo/actions.ts"
import {fill} from "./promo/fill.ts"
import {
    FindPlayersRequest,
    FindPlayersResponse,
    GetReplayRequest,
    GetReplayResponse,
    GetRulesRequest,
    GetRulesResponse,
    GrantChargesRequest,
    GrantChargesResponse,
    Player,
    ReleaseAllegianceRequest,
    ReleaseAllegianceResponse,
    RevertPlayerRequest,
    RevertPlayerResponse,
    RuleSet,
    SaveRuleSetRequest,
    SaveRuleSetResponse,
    SwitchBoardRequest,
    SwitchBoardResponse,
    TopPlayersRequest,
    TopPlayersResponse,
} from "./gen/grpc/planet/v1/admin_pb.ts"
import {
    CreateVideoUploadRequest,
    CreateVideoUploadResponse,
    KeepVideoRequest,
    KeepVideoResponse,
    ListVideosRequest,
    ListVideosResponse,
    RemoveVideoRequest,
    RemoveVideoResponse,
    Video,
} from "./gen/grpc/clips/v1/admin_pb.ts"
import {
    ListRowsRequest,
    ListRowsResponse,
    ListSnapshotsRequest,
    ListSnapshotsResponse,
    StatsRow,
    StatsSnapshot,
    TikTokAccount,
} from "./gen/grpc/publishing/v1/admin_pb.ts"
import {
    ArrivalCount,
    ArrivalSource,
    CountArrivalsRequest,
    CountArrivalsResponse,
} from "./gen/grpc/auth/v1/admin_pb.ts"
import {GetTerritoriesRequest, GetTerritoriesResponse, Territory} from "./gen/grpc/planet/v1/internal_pb.ts"
import {Graffiti, GraffitiKind, ListGraffitiRequest, ListGraffitiResponse} from "./gen/grpc/vandalsquad/v1/admin_pb.ts"
import {GetLedgerRequest, GetLedgerResponse, LedgerChange} from "./gen/grpc/planet/v1/ledger_pb.ts"
import {BoardEra} from "./gen/grpc/planet/v1/history_pb.ts"
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
    ListRenamesRequest,
    ListRenamesResponse,
    PlayerMatch,
    Rename,
    RenamePlayerRequest,
    RenamePlayerResponse,
    SearchPlayersByEmailRequest,
    SearchPlayersByEmailResponse,
    SearchPlayersRequest,
    SearchPlayersResponse,
} from "./gen/grpc/player/v1/admin_pb.ts"
import {GetPlayersRequest, GetPlayersResponse} from "./gen/grpc/player/v1/internal_pb.ts"
import {GetRosterRequest, GetRosterResponse, Player as Identity, RosterEntry} from "./gen/grpc/player/v1/player_pb.ts"
import {
    AdminGetProgressRequest,
    AdminGetProgressResponse,
    BackfillDaysPlayedRequest,
    BackfillDaysPlayedResponse,
    ForgiveMissedDayRequest,
    ForgiveMissedDayResponse,
    ReconcileLevelsRequest,
    ReconcileLevelsResponse,
    ReconcileTitlesRequest,
    ReconcileTitlesResponse,
} from "./gen/grpc/progression/v1/admin_pb.ts"
import {CountryTiles, Stats} from "./gen/grpc/progression/v1/progression_pb.ts"
import {
    AnswerNoteRequest,
    AnswerNoteResponse,
    CloseLetterRequest,
    CloseLetterResponse,
    CloseNoteRequest,
    CloseNoteResponse,
    DecideReportRequest,
    DecideReportResponse,
    Letter,
    LetterMail,
    ListLettersRequest,
    ListLettersResponse,
    ListNotesRequest,
    ListNotesResponse,
    ListReportsRequest,
    ListReportsResponse,
    Note,
    NoteAnswer,
    NoteKind,
    Report,
    WriteLetterRequest,
    WriteLetterResponse,
} from "./gen/grpc/support/v1/admin_pb.ts"
import {threadOf, waitsOf} from "./support/letters.ts"
import {
    GetRulesRequest as MissionsGetRulesRequest,
    GetRulesResponse as MissionsGetRulesResponse,
    SaveRulesRequest,
    SaveRulesResponse,
} from "./gen/grpc/missions/v1/admin_pb.ts"
import {Rules as MissionRules} from "./gen/grpc/missions/v1/rules_pb.ts"
import {
    Flag, ListFlagsRequest, ListFlagsResponse, Picture, Pin, Rejection, RejectionReason, Sold, SubmissionState,
} from "./gen/grpc/flags/v1/flags_pb.ts"
import {
    AddTimeRequest,
    AddTimeResponse,
    ApproveSubmissionRequest,
    ApproveSubmissionResponse,
    CloseFlagRequest,
    CloseFlagResponse,
    CreateFlagUploadRequest,
    CreateFlagUploadResponse,
    KeptSubmission,
    ListSubmissionsRequest,
    ListSubmissionsResponse,
    OpenFlagRequest,
    OpenFlagResponse,
    OpenSubmittedFlagRequest,
    OpenSubmittedFlagResponse,
    RejectSubmissionRequest,
    RejectSubmissionResponse,
    RenameFlagRequest,
    RenameFlagResponse,
} from "./gen/grpc/flags/v1/admin_pb.ts"
import {
    AlertsAdmin, AntibotAdmin, AuthAdmin, BonusesAdmin, BoxesAdmin, ChatAdmin, ClipsAdmin, Clients, FlagsAdmin, GamemasterAdmin, MissionsAdmin,
    PlanetAdmin, PlayerAdmin, ProgressionAdmin, PromoAdmin, QuizzesAdmin, SupportAdmin, VandalsquadAdmin,
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
export class FakeAdmin implements ChatAdmin, PlanetAdmin, PlayerAdmin, SupportAdmin, ClipsAdmin, AuthAdmin {
    readonly kind = "fake admin"
    readonly antibot: FakeAntibot = new FakeAntibot(this)
    readonly bonuses: FakeBonuses = new FakeBonuses(this)
    readonly boxes: FakeBoxes = new FakeBoxes(this)
    readonly quizzes: FakeQuizzes = new FakeQuizzes(this)
    readonly alerts: FakeAlerts = new FakeAlerts(this)
    readonly progression: FakeProgression = new FakeProgression(this)
    readonly gamemaster: FakeGamemaster = new FakeGamemaster(this)
    readonly missions: FakeMissions = new FakeMissions(this)
    readonly promo: FakePromo = new FakePromo(this)
    readonly vandalsquad: FakeVandalsquad = new FakeVandalsquad(this)
    readonly flags: FakeFlags = new FakeFlags(this)
    readonly asked: {method: string, request: unknown}[] = []
    readonly refusals = new Map<string, Code>()
    private readonly holds = new Map<string, Promise<void>>()
    // What GetMute answers, for the whole chat and for the GIFs: no mute, unless a test sets one.
    muting = new GetMuteResponse()
    // The account's moderation history, newest first: a mute Claude gave, and a rename the operator made.
    moderation = [
        new ModerationAction({
            actionId: "2", accountId: ACCOUNT, actedAt: Timestamp.fromDate(new Date(AT.getTime() - 3600_000)),
            action: {case: "mute", value: new MuteAction({kind: MuteKind.GIFS, duration: new Duration({seconds: 900n})})},
            reason: "a gore GIF, 2nd offence", operatorAccountId: "account-operator", client: "Claude",
        }),
    ]
    renames = [
        new Rename({
            renameId: "1", accountId: ACCOUNT, renamedAt: Timestamp.fromDate(new Date(AT.getTime() - 7200_000)),
            oldName: "Hitler88", newName: "Player42", reason: "a Nazi reference", operatorAccountId: "account-operator",
        }),
    ]
    gifMuting = new GetMuteResponse()
    // Whether the rules in force lock every flag, as ReleaseAllegiance answers.
    flagsLocked = false
    // The account's Discord link, as GetPlayerLink answers; UnlinkPlayer forgets it.
    discordLink: DiscordLink | undefined = new DiscordLink({
        discordUserId: "80351110224678912", countryId: "fr", linkedAtUnixMs: BigInt(AT.getTime()),
    })
    // What LinkMembers answers.
    membersLinked = new LinkMembersResponse({linked: 7, named: 5, outside: 13})
    message = new GetMessageResponse({
        messageId: "message-1", accountId: "account-reported", body: textBody("you all suck"),
        sentAt: Timestamp.fromDate(AT), countryId: "fr",
    })
    // The chat ListMessages pages through, the reported message among it: Bob baits, Cleo answers him, Ana insults,
    // then Bob again, in Germany's room.
    chat = [
        new KeptMessage({
            messageId: "message-before-1", accountId: "account-reporter", authorName: "Bob", body: textBody("come at me"),
            sentAt: Timestamp.fromDate(new Date(AT.getTime() - 120_000)), countryId: "de",
        }),
        new KeptMessage({
            messageId: "message-before-2", accountId: "account-cleo", authorName: "Cleo", body: textBody("calm down"),
            sentAt: Timestamp.fromDate(new Date(AT.getTime() - 60_000)), countryId: "it",
            replyTo: new KeptQuote({messageId: "message-before-1", authorName: "Bob", body: textBody("come at me")}),
        }),
        new KeptMessage({
            messageId: "message-1", accountId: "account-reported", authorName: "Ana", body: textBody("you all suck"),
            sentAt: Timestamp.fromDate(AT), countryId: "fr",
        }),
        new KeptMessage({
            messageId: "message-after-1", accountId: "account-reporter", authorName: "Bob", body: textBody("so rude"),
            sentAt: Timestamp.fromDate(new Date(AT.getTime() + 60_000)), countryId: "de", flagRoom: "de",
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
    warning = new WarnPlayerResponse({accountId: ACCOUNT, warnedAt: Timestamp.fromDate(AT)})
    // The set the migration keeps, in part: the fake keeps what it is given and checks nothing.
    // The numbers missions draws with, as migration 20261106180000_rules seeds them.
    missionRules = MissionRules.fromJson({flagMissions: {goalPercent: 30, leastGoal: 250, fewestPoints: 3, mostPoints: 10}})
    antibotRules = new AntibotGetRulesResponse({
        sets: [new AntibotRuleSet({
            name: "default",
            rules: Rules.fromJson({
                bans: {enforce: true, banDurations: ["86400s", "604800s", "94608000s"], reflagInterval: "300s"},
                jury: {minSuspects: 2, suspicionWindow: "600s"},
                retaker: {enabled: true}, sequencer: {enabled: true}, metronome: {enabled: true},
                defender: {enabled: false}, catcher: {enabled: true}, cohort: {enabled: true, minMembers: 2, certainMembers: 6},
                scraper: {enabled: true}, churner: {enabled: true}, hopper: {enabled: true}, rotator: {enabled: true},
                sharing: {enabled: true, window: "3600s", maxAccounts: 3},
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
    identities = [new Identity({accountId: ACCOUNT, name: "BraveFox42"})]
    profile = new AdminGetPlayerResponse({name: "BraveFox42", linked: true, createdAtUnixMs: BigInt(AT.getTime())})
    progress = new AdminGetProgressResponse({
        messagesSent: 4n, stats: new Stats({tilesTaken: 1200n, streakCurrent: 3, streakBest: 12}),
        playsFor: [new CountryTiles({countryId: "fr", tiles: 1100n}), new CountryTiles({countryId: "gb", tiles: 100n})],
    })
    // A bug from a named account, waiting for an answer, and an idea whose sender deleted its account.
    notes = [
        new Note({
            id: "note-bug", kind: NoteKind.BUG, accountId: ACCOUNT, text: "the globe goes black",
            submittedAt: Timestamp.fromDate(AT), replyAddress: "ada@example.com",
        }),
        new Note({
            id: "note-idea", kind: NoteKind.SUGGESTION, accountId: "", text: "more flags",
            submittedAt: Timestamp.fromDate(new Date(AT.getTime() - 60_000)),
        }),
    ]

    // What the operator wrote to players first: none yet. Only BraveFox42 holds a verified address.
    letters: Letter[] = []
    addresses = new Map([[ACCOUNT, "ada@example.com"]])

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
        board: "earth-1",
    })
    ledgerPages: LedgerChange[][] = []
    // A replay of the world on earth-1 since the epoch, its window the one asked.
    replay = new GetReplayResponse({eras: [new BoardEra({startedAt: Timestamp.fromDate(new Date(0)), board: "earth-1"})]})
    territories = new GetTerritoriesResponse({
        tiles: 260503,
        board: "earth-1",
        territories: [new Territory({countryId: "fr", tiles: 1200}), new Territory({countryId: "dz", tiles: 800})],
    })
    // The set the migration keeps: the game's own rules.
    rules = new GetRulesResponse({sets: [new RuleSet({name: "default", rules: PlanetRules.fromJson({
        clicks: {
            tempo: 1, perSecond: 0.2, burst: 60, scopeMultiplier: 10, v6ScopeMultiplier: 4, guestScopeMultiplier: 1,
            linkedMultiplier: 2, newAccountClicks: 10, fastStart: "1800s",
        },
        toll: {steps: [{share: 0.1, slowdown: 1.5}, {share: 0.2, slowdown: 2.5}, {share: 0.3, slowdown: 4}]},
        charges: {
            refill: {held: 1},
            spread: {clicks: 8, rings: 1, unlimited: false, off: false},
            bomb: {rings: 4, off: false, held: 1},
            enclose: {maxTiles: 25, held: 3, unlimited: false, off: false},
            shield: {held: 30, perTile: 10},
            virus: {infections: 32, every: "20s", held: 1},
            mine: {held: 3, rings: 2},
            rally: {held: 3, rings: 18, lasts: "180s", cost: 0.5, off: false},
            trap: {held: 3, clicks: 10},
        },
        frozen: false,
        allegianceLocked: false,
        arena: {on: false, freeze: "8s"},
        seed: {capitals: {shields: 100}},
        reach: false,
        capitals: {knockOut: false, fixedShields: false, cutOff: false},
    })})], playing: "default"})
    // The bonuses module's set the migration keeps: what a box caught or a right answer grants, and the sign-in refill.
    bonusRules = new BonusesGetRulesResponse({sets: [new BonusesRuleSet({name: "default", rules: BonusesRules.fromJson({
        kinds: {refill: 5, spreadClicks: 3, encloseClicks: 2, bomb: 1, shields: 3, virus: 0, mine: 0, rally: 0, trap: 0},
        gates: {
            on: false,
            open: {refill: true, spreadClicks: true, encloseClicks: false, bomb: false, shields: true, virus: false, mine: false,
                rally: false, trap: false},
        },
        spread: {maxPerBox: 4},
        enclose: {maxPerBox: 3},
        shield: {minPerBox: 5, maxPerBox: 20},
        mine: {maxPerBox: 2},
        rally: {maxPerBox: 1},
        trap: {maxPerBox: 2},
        signInRefill: true,
    })})], playing: "default"})
    // The boxes' set production plays: when a box comes, how long it flies, and a newcomer's first box.
    boxRules = new BoxesGetRulesResponse({sets: [new BoxesRuleSet({name: "default", rules: BoxesRules.fromJson({
        minInterval: "240s", maxInterval: "480s", maxBoxesPerHour: 12,
        offerTtl: "15s", missRetry: "120s", activeWithin: "120s",
        welcome: {within: "600s", boxAfter: "60s"},
        frozen: false,
    })})], playing: "default"})
    // The quizzes' set production plays: when a quiz comes, how long it stays and how its question is drawn.
    quizRules = new QuizzesGetRulesResponse({sets: [new QuizzesRuleSet({name: "default", rules: QuizzesRules.fromJson({
        enabled: true, minInterval: "240s", maxInterval: "480s", missRetry: "120s", offerTtl: "25s", answerWindow: "8s",
        leaderBias: 1, maxRightAnswersPerHour: 12,
        target: 0.7, width: 1, floor: 0.42, floorAfter: 15, explore: 0.1,
        activeWithin: "120s",
        welcome: {within: "600s"},
        frozen: false,
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
    // What CountArrivals answers, whatever the window asked.
    arrivals = [
        new ArrivalCount({day: "2026-10-17", source: ArrivalSource.TIKTOK, accounts: 4}),
        new ArrivalCount({day: "2026-10-18", source: ArrivalSource.TIKTOK, accounts: 6}),
        new ArrivalCount({day: "2026-10-18", source: ArrivalSource.YOUTUBE, accounts: 3}),
        new ArrivalCount({day: "2026-10-18", source: ArrivalSource.UNKNOWN, accounts: 20}),
    ]
    // The videos kept, the newest first, and the upload each link was made for, by video id.
    videos = [
        new Video({
            videoId: "video-1", name: "season-1-france", caption: "France held the map to the end. #clickplanet",
            contentType: "video/mp4", sizeBytes: 12_582_912n, keptAt: Timestamp.fromDate(AT),
            viewUrl: "https://bucket.test/video-1/view", downloadUrl: "https://bucket.test/video-1/download",
        }),
    ]
    readonly uploads = new Map<string, PartialMessage<CreateVideoUploadRequest>>()
    // The swastikas the vandal squad flagged, newest first: one of BraveFox42's, on the first five tiles.
    graffiti = [
        new Graffiti({
            board: "earth-1", accountId: ACCOUNT, countryId: "de", kind: GraffitiKind.SWASTIKA, tileIds: [1, 2, 3, 4, 5],
            flaggedAt: Timestamp.fromDate(AT), seenAt: Timestamp.fromDate(new Date(AT.getTime() + 600_000)),
            link: "https://clickplanet.lol/play?lat=48.85&lon=2.35&zoom=20.0",
        }),
    ]

    clients(): Clients {
        return {
            alerts: this.alerts, antibot: this.antibot, bonuses: this.bonuses, boxes: this.boxes, chat: this, planet: this, player: this, progression: this.progression,
            gamemaster: this.gamemaster, missions: this.missions, support: this, clips: this, promo: this.promo,
            publishing: this, quizzes: this.quizzes, vandalsquad: this.vandalsquad, auth: this, flags: this.flags,
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

    countArrivals(request: PartialMessage<CountArrivalsRequest>): Promise<CountArrivalsResponse> {
        return this.answer("countArrivals", request, new CountArrivalsResponse({counts: this.arrivals}))
    }

    createVideoUpload(request: PartialMessage<CreateVideoUploadRequest>): Promise<CreateVideoUploadResponse> {
        const videoId = `upload-${this.uploads.size + 1}`
        const answer = this.answer("createVideoUpload", request,
            new CreateVideoUploadResponse({videoId, uploadUrl: `https://bucket.test/${videoId}`}))
        if (!this.refusals.has("createVideoUpload")) this.uploads.set(videoId, request)
        return answer
    }

    // Kept as the server keeps it: listed first. The fake has no bucket, so any link made is kept.
    keepVideo(request: PartialMessage<KeepVideoRequest>): Promise<KeepVideoResponse> {
        const answer = this.answer("keepVideo", request, new KeepVideoResponse())
        const upload = this.uploads.get(request.videoId ?? "")
        if (!this.refusals.has("keepVideo") && upload !== undefined) {
            const link = `https://bucket.test/${request.videoId}`
            this.videos = [new Video({
                videoId: request.videoId, name: request.name, caption: request.caption, contentType: upload.contentType,
                sizeBytes: upload.sizeBytes, keptAt: Timestamp.fromDate(AT), viewUrl: `${link}/view`,
                downloadUrl: `${link}/download`,
            }), ...this.videos]
        }
        return answer
    }

    listVideos(request: PartialMessage<ListVideosRequest>): Promise<ListVideosResponse> {
        return this.answer("listVideos", request, new ListVideosResponse({videos: this.videos}))
    }

    removeVideo(request: PartialMessage<RemoveVideoRequest>): Promise<RemoveVideoResponse> {
        const answer = this.answer("removeVideo", request, new RemoveVideoResponse())
        if (!this.refusals.has("removeVideo")) this.videos = this.videos.filter(video => video.videoId !== request.videoId)
        return answer
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

    unmute(request: PartialMessage<UnmuteRequest>): Promise<UnmuteResponse> {
        return this.answer("unmute", request, new UnmuteResponse({mutedUntil: Timestamp.fromDate(AT)}))
    }

    listModeration(request: PartialMessage<ListModerationRequest>): Promise<ListModerationResponse> {
        return this.answer("listModeration", request, new ListModerationResponse({actions: this.moderation}))
    }

    listRenames(request: PartialMessage<ListRenamesRequest>): Promise<ListRenamesResponse> {
        return this.answer("listRenames", request, new ListRenamesResponse({renames: this.renames}))
    }

    getMute(request: PartialMessage<GetMuteRequest>): Promise<GetMuteResponse> {
        return this.answer("getMute", request, request.kind === MuteKind.GIFS ? this.gifMuting : this.muting)
    }

    revertPlayer(request: PartialMessage<RevertPlayerRequest>): Promise<RevertPlayerResponse> {
        const restored = request.dryRun === true ? 0 : 118
        return this.answer("revertPlayer", request, new RevertPlayerResponse({touched: 900, held: 120, restored, shields: 40}))
    }

    grantCharges(request: PartialMessage<GrantChargesRequest>): Promise<GrantChargesResponse> {
        return this.answer("grantCharges", request, new GrantChargesResponse({
            accountId: request.accountId,
            before: new ChargesHeld({enclosures: 2}),
            after: new ChargesHeld({bombs: 2, enclosures: 3}),
        }))
    }

    releaseAllegiance(request: PartialMessage<ReleaseAllegianceRequest>): Promise<ReleaseAllegianceResponse> {
        return this.answer("releaseAllegiance", request, new ReleaseAllegianceResponse({locked: this.flagsLocked}))
    }

    topPlayers(request: PartialMessage<TopPlayersRequest>): Promise<TopPlayersResponse> {
        return this.answer("topPlayers", request, new TopPlayersResponse({players: this.players, total: 75}))
    }

    findPlayers(request: PartialMessage<FindPlayersRequest>): Promise<FindPlayersResponse> {
        return this.answer("findPlayers", request, new FindPlayersResponse({players: this.players, total: 2}))
    }

    getReplay(request: PartialMessage<GetReplayRequest>): Promise<GetReplayResponse> {
        const answer = this.replay.clone()
        answer.since = new Timestamp(request.since ?? Timestamp.fromDate(AT))
        answer.until = new Timestamp(request.until ?? Timestamp.fromDate(AT))
        return this.answer("getReplay", request, answer)
    }

    // The ledger's first page, then each of ledgerPages, all on the ledger's board (earth-1 when it names none).
    getLedger(request: PartialMessage<GetLedgerRequest>): Promise<GetLedgerResponse> {
        const pages = [this.ledger.changes, ...this.ledgerPages]
        const index = request.page ? Number(request.page) : 0
        const next = index + 1 < pages.length ? String(index + 1) : ""
        const board = this.ledger.board || "earth-1"
        return this.answer("getLedger", request, new GetLedgerResponse({changes: pages[index] ?? [], nextPage: next, board}))
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
            this.rules = new GetRulesResponse({
                sets: setsWith(this.rules.sets, new RuleSet(request.set)),
                playing: this.rules.playing,
            })
        }
        return answer
    }

    // No page switches the world's board: the operator does it through Claude (docs/boards.md, the runbook).
    switchBoard(request: PartialMessage<SwitchBoardRequest>): Promise<SwitchBoardResponse> {
        return this.answer("switchBoard", request, new SwitchBoardResponse({from: "earth-1", to: request.board ?? ""}))
    }

    getRoster(request: PartialMessage<GetRosterRequest>): Promise<GetRosterResponse> {
        return this.answer("getRoster", request, new GetRosterResponse({entries: this.roster}))
    }

    getPlayers(request: PartialMessage<GetPlayersRequest>): Promise<GetPlayersResponse> {
        const asked = new Set(request.accountIds)
        return this.answer("getPlayers", request,
            new GetPlayersResponse({players: this.identities.filter(identity => asked.has(identity.accountId))}))
    }

    // The accounts the fake names, matched as the server does: in any case, anywhere in the name.
    searchPlayers(request: PartialMessage<SearchPlayersRequest>): Promise<SearchPlayersResponse> {
        const name = (request.name ?? "").trim().toLowerCase()
        const players = this.identities
            .filter(identity => identity.name.toLowerCase().includes(name))
            .map(identity => new PlayerMatch({accountId: identity.accountId, name: identity.name, guest: identity.guest}))
        return this.answer("searchPlayers", request, new SearchPlayersResponse({players, total: players.length}))
    }

    // The fake's players hold no address, and no page asks it yet.
    searchPlayersByEmail(request: PartialMessage<SearchPlayersByEmailRequest>): Promise<SearchPlayersByEmailResponse> {
        return this.answer("searchPlayersByEmail", request, new SearchPlayersByEmailResponse())
    }

    getPlayer(request: PartialMessage<AdminGetPlayerRequest>): Promise<AdminGetPlayerResponse> {
        return this.answer("getPlayer", request, this.profile)
    }

    renamePlayer(request: PartialMessage<RenamePlayerRequest>): Promise<RenamePlayerResponse> {
        const name = (request.name ?? "").trim()
        if (!this.refusals.has("renamePlayer")) this.profile = new AdminGetPlayerResponse({...this.profile, name})
        return this.answer("renamePlayer", request, new RenamePlayerResponse({name}))
    }

    listNotes(request: PartialMessage<ListNotesRequest>): Promise<ListNotesResponse> {
        const kind = request.kind ?? NoteKind.UNSPECIFIED
        const notes = this.notes.filter(note => (kind === NoteKind.UNSPECIFIED || note.kind === kind)
            && (request.openOnly !== true || waiting(note)))
        return this.answer("listNotes", request, new ListNotesResponse({notes}))
    }

    // Kept as the server keeps it, unless refused.
    answerNote(request: PartialMessage<AnswerNoteRequest>): Promise<AnswerNoteResponse> {
        const sentAt = Timestamp.fromDate(this.now())
        const answer = this.answer("answerNote", request, new AnswerNoteResponse({sentAt}))
        if (!this.refusals.has("answerNote")) {
            this.notes = this.notes.map(note => note.id !== request.noteId ? note : new Note({
                ...note, answers: [...note.answers, new NoteAnswer({text: request.text, sentAt})],
            }))
        }
        return answer
    }

    closeNote(request: PartialMessage<CloseNoteRequest>): Promise<CloseNoteResponse> {
        const closedAt = Timestamp.fromDate(this.now())
        const answer = this.answer("closeNote", request, new CloseNoteResponse({closedAt}))
        if (!this.refusals.has("closeNote")) {
            this.notes = this.notes.map(note => note.id !== request.noteId ? note : new Note({...note, closedAt}))
        }
        return answer
    }

    // The letter about a submission, to the account's verified address, or one more mail in a letter, refused as the server
    // refuses it.
    writeLetter(request: PartialMessage<WriteLetterRequest>): Promise<WriteLetterResponse> {
        const text = (request.text ?? "").trim()
        const sentAt = Timestamp.fromDate(this.now())
        const refused = (why: string, code: Code) => this.answer("writeLetter", request, undefined).then((): WriteLetterResponse => {
            throw new ConnectError(why, code)
        })
        if (text === "" || [...text].length > 5000) return refused("a text of 1 to 5,000 characters", Code.InvalidArgument)
        let letter: Letter
        if ((request.letterId ?? "") !== "") {
            const kept = this.letters.find(each => each.id === request.letterId)
            if (kept === undefined) return refused("no such letter", Code.NotFound)
            letter = new Letter({...kept, mails: [...kept.mails, new LetterMail({text, sentAt})]})
        } else {
            const submissionId = request.submissionId ?? ""
            if (submissionId === "") return refused("a letter is about a flag submission", Code.InvalidArgument)
            if (this.letters.some(each => each.submissionId === submissionId)) {
                return refused("the submission has its letter already", Code.AlreadyExists)
            }
            const address = this.addresses.get(request.accountId ?? "")
            if (address === undefined) return refused("the account has no verified address", Code.FailedPrecondition)
            const subject = (request.subject ?? "").trim()
            if (subject === "" || [...subject].length > 200) return refused("a subject of 1 to 200 characters", Code.InvalidArgument)
            letter = new Letter({
                id: `letter-${this.letters.length + 1}`, submissionId, accountId: request.accountId, address, subject,
                mails: [new LetterMail({text, sentAt})],
            })
        }
        const answer = this.answer("writeLetter", request, new WriteLetterResponse({letter}))
        if (!this.refusals.has("writeLetter")) this.letters = [letter, ...this.letters.filter(each => each.id !== letter.id)]
        return answer
    }

    listLetters(request: PartialMessage<ListLettersRequest>): Promise<ListLettersResponse> {
        const letters = latestLettersFirst(this.letters.filter(letter =>
            ((request.accountId ?? "") === "" || letter.accountId === request.accountId) &&
            ((request.submissionId ?? "") === "" || letter.submissionId === request.submissionId) &&
            (request.openOnly !== true || waitsOf(letter))))
        return this.answer("listLetters", request, new ListLettersResponse({letters: letters.slice(0, Math.min(request.limit || 100, 500))}))
    }

    closeLetter(request: PartialMessage<CloseLetterRequest>): Promise<CloseLetterResponse> {
        const kept = this.letters.find(each => each.id === request.letterId)
        if (kept === undefined) {
            return this.answer("closeLetter", request, undefined).then((): CloseLetterResponse => {
                throw new ConnectError("no such letter", Code.NotFound)
            })
        }
        const closedAt = Timestamp.fromDate(this.now())
        const answer = this.answer("closeLetter", request, new CloseLetterResponse({closedAt}))
        if (!this.refusals.has("closeLetter")) {
            this.letters = this.letters.map(letter => letter.id === kept.id ? new Letter({...letter, closedAt}) : letter)
        }
        return answer
    }

    // When an answer is sent or a note closed: the demo's is the clock's.
    protected now(): Date {
        return AT
    }

    listReports(request: PartialMessage<ListReportsRequest>): Promise<ListReportsResponse> {
        return this.answer("listReports", request, new ListReportsResponse({reports: this.reports}))
    }

    decideReport(request: PartialMessage<DecideReportRequest>): Promise<DecideReportResponse> {
        return this.answer("decideReport", request, new DecideReportResponse({decidedAt: Timestamp.fromDate(AT)}))
    }
}

const DAY_MS = 86_400_000

// A flag opened since the game shipped, as the server lists it; closed when it has a close.
export function soldFlag(
    code: string, name: string, endsAt: Date, closedAt?: Date, url = `https://media.test/flags/pictures/${code}.png`,
    pin: {latitude: number, longitude: number} = {latitude: 0, longitude: 0},
): Flag {
    return new Flag({
        code, name, closed: closedAt !== undefined,
        sold: new Sold({
            picture: new Picture({url, width: 768, height: 512, focusX: 0.5, focusY: 0.5}),
            endsAt: Timestamp.fromDate(endsAt),
            closedAt: closedAt === undefined ? undefined : Timestamp.fromDate(closedAt),
            pin: new Pin(pin),
        }),
    })
}

export const MUNICH = {latitude: 48.1374, longitude: 11.5755}
export const CHAMBERY = {latitude: 45.5646, longitude: 5.9178}

// A submission as the server keeps it: waiting, from BraveFox42, with a private picture, unless `part` says otherwise.
export function keptSubmission(
    id: string, name: string, pin: {latitude: number, longitude: number}, part: PartialMessage<KeptSubmission> = {},
): KeptSubmission {
    const submittedAt = Timestamp.fromDate(new Date(AT.getTime() - 3_600_000))
    return new KeptSubmission({
        id, accountId: ACCOUNT, name, pin: new Pin(pin), state: SubmissionState.WAITING, submittedAt, decidedAt: submittedAt,
        picture: new Picture({url: `https://bucket.test/flags/submissions/${id}.png?signed`, width: 768, height: 512}),
        otherColorsPercent: 2, ...part,
    })
}

function onGlobe(pin: PartialMessage<Pin>): boolean {
    return Math.abs(pin.latitude ?? 0) <= 90 && Math.abs(pin.longitude ?? 0) <= 180
}

// The server checks all 235 flags the game ships with, where the fake lists a few: the rest by their region names.
const GAME_FLAG_NAMES: readonly string[] = (() => {
    const regions = new Intl.DisplayNames(["en"], {type: "region", fallback: "none"})
    const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    const names: string[] = []
    for (const first of letters) {
        for (const second of letters) {
            const name = regions.of(first + second)
            if (name !== undefined) names.push(name)
        }
    }
    return names
})()

// What the server refuses of a flag's name: 1 to 40 characters, and no other flag's, built-in or sold, whatever its case.
export function nameRefusalOf(name: string, others: readonly Flag[]): string | undefined {
    if (name === "" || [...name].length > 40) return "a name of 1 to 40 characters"
    const taken = others.find(other => other.name.toLowerCase() === name.toLowerCase())
    if (taken !== undefined) return `the flag ${taken.code} is named ${taken.name}`
    const game = GAME_FLAG_NAMES.find(other => other.toLowerCase() === name.toLowerCase())
    return game === undefined ? undefined : `a flag the game ships with is named ${game}`
}

// flags.v1's calls, asked of the same fake: the game's own flags the made-up players play for, then the ones opened
// since, by name: Bavaria, open 12 days more after AT, and Savoy, closed 3 days before it; and the submissions, the
// waiting ones oldest first, then the decided ones latest first. Each change is kept as the server keeps it, at the
// time of the call, and refused as the server refuses it.
// The latest written or replied to first, as the server lists them.
function latestLettersFirst(letters: Letter[]): Letter[] {
    const last = (letter: Letter) => threadOf(letter).filter(entry => entry.kind !== "closed").at(-1)?.at?.toDate().getTime() ?? 0
    return [...letters].sort((a, b) => last(b) - last(a))
}

export class FakeFlags implements FlagsAdmin {
    listed = [
        ...[["fr", "France"], ["dz", "Algeria"], ["de", "Germany"], ["it", "Italy"], ["gb", "UK"]]
            .map(([code, name]) => new Flag({code, name})),
        soldFlag("r-k3p9x2ma0b", "Bavaria", new Date(AT.getTime() + 12 * DAY_MS), undefined, undefined, MUNICH),
        soldFlag("r-0a1b2c3d4e", "Savoy", new Date(AT.getTime() + 4 * DAY_MS), new Date(AT.getTime() - 3 * DAY_MS), undefined, CHAMBERY),
    ]
    // Catalonia, fine; a photo of a lake, from an account the player read names none; a new name and picture for
    // Bavaria; Sardinia, approved; Brittany, rejected as the game has it; and Savoy, opened from its submission.
    submissions = [
        keptSubmission("sub-catalonia", "Catalonia", {latitude: 41.3874, longitude: 2.1686}, {
            submittedAt: Timestamp.fromDate(new Date(AT.getTime() - 5 * 3_600_000)),
        }),
        keptSubmission("sub-beach", "Lake Garda", {latitude: 45.6, longitude: 10.6}, {otherColorsPercent: 41, accountId: "account-gone"}),
        keptSubmission("sub-bavaria", "Free State of Bavaria", {latitude: 48.1374, longitude: 11.5755}, {
            change: true, flag: "r-k3p9x2ma0b",
        }),
        keptSubmission("sub-sardinia", "Sardinia", {latitude: 39.2238, longitude: 9.1217}, {
            state: SubmissionState.APPROVED, decidedAt: Timestamp.fromDate(new Date(AT.getTime() - DAY_MS)),
            picture: new Picture({url: "https://media.test/flags/pictures/sardinia.png", width: 768, height: 512}),
        }),
        keptSubmission("sub-brittany", "Brittany", {latitude: 48.1173, longitude: -1.6778}, {
            state: SubmissionState.REJECTED, decidedAt: Timestamp.fromDate(new Date(AT.getTime() - 2 * DAY_MS)),
            rejection: new Rejection({reason: RejectionReason.EXISTS}),
        }),
        keptSubmission("sub-savoy", "Savoy", CHAMBERY, {
            state: SubmissionState.OPENED, flag: "r-0a1b2c3d4e", decidedAt: Timestamp.fromDate(new Date(AT.getTime() - 40 * DAY_MS)),
            picture: new Picture({url: "https://media.test/flags/pictures/r-0a1b2c3d4e.png", width: 768, height: 512}),
        }),
    ]
    private readonly uploads = new Map<string, PartialMessage<CreateFlagUploadRequest>>()

    constructor(private readonly admin: FakeAdmin) {}

    listFlags(request: PartialMessage<ListFlagsRequest>): Promise<ListFlagsResponse> {
        return this.admin.answer("listFlags", request, new ListFlagsResponse({flags: this.listed}))
    }

    createFlagUpload(request: PartialMessage<CreateFlagUploadRequest>): Promise<CreateFlagUploadResponse> {
        const uploadId = `flag-upload-${this.uploads.size + 1}`
        const answer = this.admin.answer("createFlagUpload", request, new CreateFlagUploadResponse({
            uploadId, uploadUrl: `https://bucket.test/flags/uploads/${uploadId}`,
        }))
        if (!this.admin.refusals.has("createFlagUpload")) this.uploads.set(uploadId, request)
        return answer
    }

    // The fake has no bucket: any upload it made a link for is a picture.
    openFlag(request: PartialMessage<OpenFlagRequest>): Promise<OpenFlagResponse> {
        if (!this.uploads.has(request.uploadId ?? "")) {
            return this.refused("openFlag", request, "no picture uploaded under this id", Code.InvalidArgument)
        }
        const pin = request.pin
        if (pin === undefined || !onGlobe(pin)) {
            return this.refused("openFlag", request, "a pin from -90 to 90 and -180 to 180", Code.InvalidArgument)
        }
        const name = (request.name ?? "").trim()
        const taken = nameRefusalOf(name, this.listed)
        if (taken !== undefined) return this.refused("openFlag", request, taken, Code.InvalidArgument)
        const flag = soldFlag(this.nextCode(), name, new Timestamp(request.endsAt).toDate(), undefined,
            undefined, {latitude: pin.latitude ?? 0, longitude: pin.longitude ?? 0})
        const answer = this.admin.answer("openFlag", request, new OpenFlagResponse({flag}))
        if (!this.admin.refusals.has("openFlag")) this.add(flag)
        return answer
    }

    // An open flag ends that much later; one closed less than 30 days ago opens again, and ends that much from now.
    addTime(request: PartialMessage<AddTimeRequest>): Promise<AddTimeResponse> {
        const flag = this.listed.find(each => each.code === request.code && each.sold !== undefined)
        if (flag === undefined) return this.refused("addTime", request, "no flag opened under this code", Code.NotFound)
        const now = new Date()
        const closedAt = flag.sold?.closedAt?.toDate()
        if (closedAt !== undefined && now.getTime() - closedAt.getTime() >= 30 * DAY_MS) {
            return this.refused("addTime", request, "closed over 30 days ago", Code.FailedPrecondition)
        }
        const from = closedAt === undefined ? flag.sold?.endsAt?.toDate() ?? now : now
        const added = soldFlag(flag.code, flag.name, new Date(from.getTime() + (request.days ?? 0) * DAY_MS), undefined,
            flag.sold?.picture?.url, flag.sold?.pin)
        return this.replaced("addTime", request, added, new AddTimeResponse({flag: added}))
    }

    closeFlag(request: PartialMessage<CloseFlagRequest>): Promise<CloseFlagResponse> {
        const flag = this.listed.find(each => each.code === request.code && each.sold !== undefined)
        if (flag === undefined) return this.refused("closeFlag", request, "no flag opened under this code", Code.NotFound)
        if (flag.closed) return this.refused("closeFlag", request, "the flag is closed already", Code.FailedPrecondition)
        if ((request.note ?? "").trim() === "") return this.refused("closeFlag", request, "an early close has a note", Code.InvalidArgument)
        const closed = soldFlag(flag.code, flag.name, flag.sold?.endsAt?.toDate() ?? new Date(), new Date(),
            flag.sold?.picture?.url, flag.sold?.pin)
        return this.replaced("closeFlag", request, closed, new CloseFlagResponse({flag: closed}))
    }

    listSubmissions(request: PartialMessage<ListSubmissionsRequest>): Promise<ListSubmissionsResponse> {
        const time = (submission: KeptSubmission) => submission.decidedAt?.toDate().getTime() ?? 0
        const waiting = this.submissions.filter(each => each.state === SubmissionState.WAITING)
            .sort((a, b) => (a.submittedAt?.toDate().getTime() ?? 0) - (b.submittedAt?.toDate().getTime() ?? 0))
        const decided = this.submissions.filter(each => each.state !== SubmissionState.WAITING).sort((a, b) => time(b) - time(a))
        return this.admin.answer("listSubmissions", request, new ListSubmissionsResponse({submissions: [...waiting, ...decided]}))
    }

    // Its picture made public, at the capital the operator moved it to or the player's; a change is on its flag at once,
    // its capital too.
    approveSubmission(request: PartialMessage<ApproveSubmissionRequest>): Promise<ApproveSubmissionResponse> {
        const submission = this.submissions.find(each => each.id === request.submissionId)
        if (submission === undefined) return this.refused("approveSubmission", request, "no such submission", Code.NotFound)
        const moved = request.pin
        if (moved !== undefined && !onGlobe(moved)) {
            return this.refused("approveSubmission", request, "a pin from -90 to 90 and -180 to 180", Code.InvalidArgument)
        }
        if (submission.state !== SubmissionState.WAITING) {
            return this.refused("approveSubmission", request, "the submission is decided already", Code.FailedPrecondition)
        }
        const taken = nameRefusalOf(submission.name, this.listed.filter(each => !submission.change || each.code !== submission.flag))
        if (taken !== undefined) return this.refused("approveSubmission", request, taken, Code.FailedPrecondition)
        const pin = moved === undefined ? submission.pin : new Pin({latitude: moved.latitude ?? 0, longitude: moved.longitude ?? 0})
        const approved = this.decided(submission, {state: SubmissionState.APPROVED, pin})
        const answer = this.admin.answer("approveSubmission", request, new ApproveSubmissionResponse({submission: approved}))
        if (!this.admin.refusals.has("approveSubmission")) {
            this.keep(approved)
            const flag = this.listed.find(each => each.code === submission.flag)
            if (submission.change && flag !== undefined) {
                this.listed = this.listed.map(each => each !== flag ? each : soldFlag(flag.code, submission.name,
                    flag.sold?.endsAt?.toDate() ?? new Date(), flag.sold?.closedAt?.toDate(), submission.picture?.url, pin))
            }
        }
        return answer
    }

    rejectSubmission(request: PartialMessage<RejectSubmissionRequest>): Promise<RejectSubmissionResponse> {
        const submission = this.submissions.find(each => each.id === request.submissionId)
        if (submission === undefined) return this.refused("rejectSubmission", request, "no such submission", Code.NotFound)
        if (submission.state !== SubmissionState.WAITING) {
            return this.refused("rejectSubmission", request, "the submission is decided already", Code.FailedPrecondition)
        }
        const reason = request.rejection?.reason ?? RejectionReason.UNSPECIFIED
        const note = (request.rejection?.note ?? "").trim()
        if (reason === RejectionReason.UNSPECIFIED) return this.refused("rejectSubmission", request, "a reason", Code.InvalidArgument)
        if (reason === RejectionReason.OTHER && note === "") {
            return this.refused("rejectSubmission", request, "a note for the reason OTHER", Code.InvalidArgument)
        }
        if ([...note].length > 500) return this.refused("rejectSubmission", request, "a note of 500 characters at most", Code.InvalidArgument)
        const rejected = this.decided(submission, {state: SubmissionState.REJECTED, rejection: new Rejection({reason, note})})
        const answer = this.admin.answer("rejectSubmission", request, new RejectSubmissionResponse({submission: rejected}))
        if (!this.admin.refusals.has("rejectSubmission")) this.keep(rejected)
        return answer
    }

    // Opened at the submission's pin, with its picture.
    openSubmittedFlag(request: PartialMessage<OpenSubmittedFlagRequest>): Promise<OpenSubmittedFlagResponse> {
        const submission = this.submissions.find(each => each.id === request.submissionId)
        if (submission === undefined) return this.refused("openSubmittedFlag", request, "no such submission", Code.NotFound)
        if (submission.state !== SubmissionState.APPROVED || submission.change) {
            return this.refused("openSubmittedFlag", request, "the submission is no approved new flag", Code.FailedPrecondition)
        }
        const taken = nameRefusalOf(submission.name, this.listed)
        if (taken !== undefined) return this.refused("openSubmittedFlag", request, taken, Code.FailedPrecondition)
        const endsAt = new Timestamp(request.endsAt).toDate()
        if (endsAt.getTime() <= Date.now()) return this.refused("openSubmittedFlag", request, "an end to come", Code.InvalidArgument)
        const flag = soldFlag(this.nextCode(), submission.name, endsAt, undefined, submission.picture?.url, submission.pin)
        const answer = this.admin.answer("openSubmittedFlag", request, new OpenSubmittedFlagResponse({flag}))
        if (!this.admin.refusals.has("openSubmittedFlag")) {
            this.add(flag)
            this.keep(this.decided(submission, {state: SubmissionState.OPENED, flag: flag.code}))
        }
        return answer
    }

    private decided(submission: KeptSubmission, part: PartialMessage<KeptSubmission>): KeptSubmission {
        return new KeptSubmission({...submission, decidedAt: Timestamp.fromDate(new Date()), ...part})
    }

    private keep(submission: KeptSubmission) {
        this.submissions = this.submissions.map(each => each.id === submission.id ? submission : each)
    }

    private nextCode(): string {
        return `r-${this.listed.length.toString(36).padStart(10, "0")}`
    }

    // A sold flag, among the others by name.
    private add(flag: Flag) {
        const sold = [...this.listed.filter(each => each.sold !== undefined), flag].sort((a, b) => a.name.localeCompare(b.name))
        this.listed = [...this.listed.filter(each => each.sold === undefined), ...sold]
    }

    // Open or closed, its code, picture and land kept.
    renameFlag(request: PartialMessage<RenameFlagRequest>): Promise<RenameFlagResponse> {
        const flag = this.listed.find(each => each.code === request.code && each.sold !== undefined)
        if (flag === undefined) return this.refused("renameFlag", request, "no flag opened under this code", Code.NotFound)
        const name = (request.name ?? "").trim()
        const refused = nameRefusalOf(name, this.listed.filter(each => each.code !== flag.code))
        if (refused !== undefined) return this.refused("renameFlag", request, refused, Code.InvalidArgument)
        const renamed = new Flag({...flag, name})
        return this.replaced("renameFlag", request, renamed, new RenameFlagResponse({flag: renamed}))
    }

    private replaced<T>(method: string, request: unknown, flag: Flag, response: T): Promise<T> {
        const answer = this.admin.answer(method, request, response)
        if (!this.admin.refusals.has(method)) this.listed = this.listed.map(each => each.code === flag.code ? flag : each)
        return answer
    }

    // Refused as the server refuses it, after the same wait as an answer.
    private refused<T>(method: string, request: unknown, why: string, code: Code): Promise<T> {
        return this.admin.answer(method, request, undefined).then((): T => {
            throw new ConnectError(why, code)
        })
    }
}

// vandalsquad.v1's calls, asked of the same fake.
export class FakeVandalsquad implements VandalsquadAdmin {
    constructor(private readonly admin: FakeAdmin) {}

    listGraffiti(request: PartialMessage<ListGraffitiRequest>): Promise<ListGraffitiResponse> {
        return this.admin.answer("listGraffiti", request, new ListGraffitiResponse({graffiti: this.admin.graffiti}))
    }
}

// alerts.v1's calls, asked of the same fake.
export class FakeAlerts implements AlertsAdmin {
    constructor(private readonly admin: FakeAdmin) {}

    getPlayerLink(request: PartialMessage<GetPlayerLinkRequest>): Promise<GetPlayerLinkResponse> {
        return this.admin.answer("getPlayerLink", request, new GetPlayerLinkResponse({link: this.admin.discordLink}))
    }

    unlinkPlayer(request: PartialMessage<UnlinkPlayerRequest>): Promise<UnlinkPlayerResponse> {
        const unlinked = this.admin.discordLink !== undefined
        const answer = this.admin.answer("unlinkPlayer", request, new UnlinkPlayerResponse({unlinked}))
        if (!this.admin.refusals.has("unlinkPlayer")) this.admin.discordLink = undefined
        return answer
    }

    linkMembers(request: PartialMessage<LinkMembersRequest>): Promise<LinkMembersResponse> {
        return this.admin.answer("linkMembers", request, this.admin.membersLinked)
    }
}

// progression.v1's calls, asked of the same fake.
export class FakeProgression implements ProgressionAdmin {
    constructor(private readonly admin: FakeAdmin) {}

    reconcileTitles(request: PartialMessage<ReconcileTitlesRequest>): Promise<ReconcileTitlesResponse> {
        return this.admin.answer("reconcileTitles", request, new ReconcileTitlesResponse({granted: 12, revoked: 3}))
    }

    backfillDaysPlayed(request: PartialMessage<BackfillDaysPlayedRequest>): Promise<BackfillDaysPlayedResponse> {
        return this.admin.answer("backfillDaysPlayed", request, new BackfillDaysPlayedResponse({accounts: 1200, changed: 37}))
    }

    reconcileLevels(request: PartialMessage<ReconcileLevelsRequest>): Promise<ReconcileLevelsResponse> {
        return this.admin.answer("reconcileLevels", request, new ReconcileLevelsResponse({raised: 2700, removed: 3}))
    }

    getProgress(request: PartialMessage<AdminGetProgressRequest>): Promise<AdminGetProgressResponse> {
        return this.admin.answer("getProgress", request, this.admin.progress)
    }

    // No page calls it yet: the operator forgives a day through the connector.
    forgiveMissedDay(request: PartialMessage<ForgiveMissedDayRequest>): Promise<ForgiveMissedDayResponse> {
        return this.admin.answer("forgiveMissedDay", request, new ForgiveMissedDayResponse({forgivenDays: [request.day ?? ""]}))
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

    warnPlayer(request: PartialMessage<WarnPlayerRequest>): Promise<WarnPlayerResponse> {
        return this.admin.answer("warnPlayer", request, this.admin.warning)
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
            this.admin.antibotRules = new AntibotGetRulesResponse({
                sets: setsWith(this.admin.antibotRules.sets, new AntibotRuleSet(request.set)),
                playing: this.admin.antibotRules.playing,
            })
        }
        return answer
    }
}

// bonuses.v1's rules, asked of the same fake as getBonusRules and saveBonusRuleSet: planet.v1 and antibot.v1 have a
// getRules and a saveRuleSet of their own.
export class FakeBonuses implements BonusesAdmin {
    constructor(private readonly admin: FakeAdmin) {}

    getRules(request: PartialMessage<BonusesGetRulesRequest>): Promise<BonusesGetRulesResponse> {
        return this.admin.answer("getBonusRules", request, this.admin.bonusRules)
    }

    // Kept as the server keeps it: a set of the same name is replaced.
    saveRuleSet(request: PartialMessage<BonusesSaveRuleSetRequest>): Promise<BonusesSaveRuleSetResponse> {
        const answer = this.admin.answer("saveBonusRuleSet", request, new BonusesSaveRuleSetResponse())
        if (!this.admin.refusals.has("saveBonusRuleSet") && request.set !== undefined) {
            this.admin.bonusRules = new BonusesGetRulesResponse({
                sets: setsWith(this.admin.bonusRules.sets, new BonusesRuleSet(request.set)),
                playing: this.admin.bonusRules.playing,
            })
        }
        return answer
    }
}

// boxes.v1's rules, asked of the same fake as getBoxRules and saveBoxRuleSet: planet.v1, antibot.v1, bonuses.v1 and
// quizzes.v1 have a getRules and a saveRuleSet of their own.
export class FakeBoxes implements BoxesAdmin {
    constructor(private readonly admin: FakeAdmin) {}

    getRules(request: PartialMessage<BoxesGetRulesRequest>): Promise<BoxesGetRulesResponse> {
        return this.admin.answer("getBoxRules", request, this.admin.boxRules)
    }

    // Kept as the server keeps it: a set of the same name is replaced.
    saveRuleSet(request: PartialMessage<BoxesSaveRuleSetRequest>): Promise<BoxesSaveRuleSetResponse> {
        const answer = this.admin.answer("saveBoxRuleSet", request, new BoxesSaveRuleSetResponse())
        if (!this.admin.refusals.has("saveBoxRuleSet") && request.set !== undefined) {
            this.admin.boxRules = new BoxesGetRulesResponse({
                sets: setsWith(this.admin.boxRules.sets, new BoxesRuleSet(request.set)),
                playing: this.admin.boxRules.playing,
            })
        }
        return answer
    }
}

// quizzes.v1's calls, asked of the same fake as getQuizRules, saveQuizRuleSet and listQuizQuestions: planet.v1,
// antibot.v1 and bonuses.v1 have a getRules and a saveRuleSet of their own.
export class FakeQuizzes implements QuizzesAdmin {
    constructor(private readonly admin: FakeAdmin) {}

    getRules(request: PartialMessage<QuizzesGetRulesRequest>): Promise<QuizzesGetRulesResponse> {
        return this.admin.answer("getQuizRules", request, this.admin.quizRules)
    }

    // Kept as the server keeps it: a set of the same name is replaced.
    saveRuleSet(request: PartialMessage<QuizzesSaveRuleSetRequest>): Promise<QuizzesSaveRuleSetResponse> {
        const answer = this.admin.answer("saveQuizRuleSet", request, new QuizzesSaveRuleSetResponse())
        if (!this.admin.refusals.has("saveQuizRuleSet") && request.set !== undefined) {
            this.admin.quizRules = new QuizzesGetRulesResponse({
                sets: setsWith(this.admin.quizRules.sets, new QuizzesRuleSet(request.set)),
                playing: this.admin.quizRules.playing,
            })
        }
        return answer
    }

    // No page reads it: Claude does, through the connector.
    listQuizQuestions(request: PartialMessage<ListQuizQuestionsRequest>): Promise<ListQuizQuestionsResponse> {
        return this.admin.answer("listQuizQuestions", request, new ListQuizQuestionsResponse())
    }
}

// The server's open_only: an address to answer at, and the player wrote last, the note or a reply after the last answer
// and the last close.
function waiting(note: Note): boolean {
    const ms = (at: Timestamp | undefined) => at?.toDate().getTime() ?? 0
    const wrote = Math.max(ms(note.submittedAt), ...note.replies.map(reply => ms(reply.receivedAt)))
    const acted = Math.max(ms(note.closedAt), ...note.answers.map(answer => ms(answer.sentAt)))
    return note.replyAddress !== "" && wrote > acted
}

// A module's sets with `saved` in place of the one of its name, default first, then by name.
function setsWith<S extends {name: string}>(sets: S[], saved: S): S[] {
    return [...sets.filter(set => set.name !== saved.name), saved]
        .sort((a, b) => Number(a.name !== "default") - Number(b.name !== "default") || a.name.localeCompare(b.name))
}

// missions.v1's rules, asked of the same fake: planet.v1 and antibot.v1 have a getRules of their own. A save replaces
// them whole, as the server does, and checks nothing.
export class FakeMissions implements MissionsAdmin {
    constructor(private readonly admin: FakeAdmin) {}

    getRules(request: PartialMessage<MissionsGetRulesRequest>): Promise<MissionsGetRulesResponse> {
        return this.admin.answer("getMissionRules", request, new MissionsGetRulesResponse({rules: this.admin.missionRules}))
    }

    saveRules(request: PartialMessage<SaveRulesRequest>): Promise<SaveRulesResponse> {
        const answer = this.admin.answer("saveMissionRules", request, new SaveRulesResponse())
        if (!this.admin.refusals.has("saveMissionRules") && request.rules !== undefined) {
            this.admin.missionRules = new MissionRules(request.rules)
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
        dailyCutoff: new Duration({seconds: BigInt(CUTOFF_HOUR * 3600)}),
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

const CUTOFF_HOUR = 21

// A season's days at a 21:00 UTC cutoff, as the server works them out; none for a season that runs for years.
function daysOf(startsAt: Date, endsAt: Date): Day[] {
    if (endsAt.getTime() - startsAt.getTime() > 120 * DAY * 1000) return []
    const days: Day[] = []
    let start = startsAt
    let end = new Date(Date.UTC(start.getUTCFullYear(), start.getUTCMonth(), start.getUTCDate(), CUTOFF_HOUR))
    if (end <= start) end = new Date(end.getTime() + DAY * 1000)
    while (end <= endsAt) {
        days.push(new Day({number: days.length + 1, startsAt: Timestamp.fromDate(start), endsAt: Timestamp.fromDate(end)}))
        start = end
        end = new Date(end.getTime() + DAY * 1000)
    }
    return days
}

// promo.v1's calls, asked of the same fake. GetCards lists the sets planet's fake keeps (`rules`), each with the cards
// `decks` holds, filled the way the page fills a draft: `{bonuses.…}` from the bonuses' fake (`bonusRules`),
// `{boxes.…}` from the boxes' (`boxRules`) and `{quizzes.…}` from the quizzes' (`quizRules`).
export class FakePromo implements PromoAdmin {
    backgrounds: PromoBackground[] = [
        new PromoBackground({
            backgroundId: "background-war", name: "The war", contentType: "image/webp", sizeBytes: 300_000n,
            keptAt: Timestamp.fromDate(AT), url: "https://media.test/promo/backgrounds/background-war",
        }),
        new PromoBackground({
            backgroundId: "background-points", name: "The points", contentType: "image/png", sizeBytes: 900_000n,
            keptAt: Timestamp.fromDate(AT), url: "https://media.test/promo/backgrounds/background-points",
        }),
    ]
    decks = new Map<string, CardDraft[]>()
    placeholders: Placeholder[] = PLACEHOLDER_PATHS.map(path => new Placeholder({path, example: ""}))
    exampleBattle = new ExampleBattle({times: 3, placeholders: Object.entries(TRIPLE).map(([path, example]) => new Placeholder({path, example}))})
    private readonly uploads = new Map<string, PartialMessage<CreateBackgroundUploadRequest>>()

    constructor(private readonly admin: FakeAdmin) {}

    createBackgroundUpload(request: PartialMessage<CreateBackgroundUploadRequest>): Promise<CreateBackgroundUploadResponse> {
        const backgroundId = `background-${this.uploads.size + 1}`
        const answer = this.admin.answer("createBackgroundUpload", request, new CreateBackgroundUploadResponse({
            backgroundId, uploadUrl: `https://bucket.test/promo/backgrounds/${backgroundId}`,
        }))
        if (!this.admin.refusals.has("createBackgroundUpload")) this.uploads.set(backgroundId, request)
        return answer
    }

    keepBackground(request: PartialMessage<KeepBackgroundRequest>): Promise<KeepBackgroundResponse> {
        const answer = this.admin.answer("keepBackground", request, new KeepBackgroundResponse())
        const upload = this.uploads.get(request.backgroundId ?? "")
        if (!this.admin.refusals.has("keepBackground") && upload !== undefined) {
            this.backgrounds = [new PromoBackground({
                backgroundId: request.backgroundId, name: request.name, contentType: upload.contentType,
                sizeBytes: upload.sizeBytes, keptAt: Timestamp.fromDate(AT), url: this.urlOf(request.backgroundId ?? ""),
            }), ...this.backgrounds]
        }
        return answer
    }

    urlOf(backgroundId: string): string {
        return `https://media.test/promo/backgrounds/${backgroundId}`
    }

    listBackgrounds(request: PartialMessage<ListBackgroundsRequest>): Promise<ListBackgroundsResponse> {
        return this.admin.answer("listBackgrounds", request, new ListBackgroundsResponse({backgrounds: this.backgrounds}))
    }

    removeBackground(request: PartialMessage<RemoveBackgroundRequest>): Promise<RemoveBackgroundResponse> {
        const answer = this.admin.answer("removeBackground", request, new RemoveBackgroundResponse())
        if (!this.admin.refusals.has("removeBackground")) {
            this.backgrounds = this.backgrounds.filter(background => background.backgroundId !== request.backgroundId)
        }
        return answer
    }

    getCards(request: PartialMessage<GetCardsRequest>): Promise<GetCardsResponse> {
        const played = playedBySet(this.admin.rules, {
            bonuses: this.admin.bonusRules, boxes: this.admin.boxRules, quizzes: this.admin.quizRules,
        })
        const rules = this.placeholders.map(placeholder => placeholder.path)
        const sets = this.admin.rules.sets.map(set => new SetCards({
            ruleSet: set.name,
            cards: (this.decks.get(set.name) ?? []).map(draft => {
                const values = {played: played.get(set.name), rules, battle: this.exampleBattle.placeholders}
                const headline = fill(draft.headline, values)
                const line = fill(draft.line, values)
                const problems = [...headline.problems, ...line.problems]
                const url = this.backgrounds.find(each => each.backgroundId === draft.backgroundId)?.url ?? ""
                return problems.length > 0
                    ? new Card({draft, problem: problems.join(" ")})
                    : new Card({draft, filled: new TeaserCard({backgroundUrl: url, headline: headline.text, line: line.text})})
            }),
        }))
        return this.admin.answer("getCards", request, new GetCardsResponse({
            sets, placeholders: this.placeholders, exampleBattle: this.exampleBattle,
        }))
    }

    saveCards(request: PartialMessage<SaveCardsRequest>): Promise<SaveCardsResponse> {
        const answer = this.admin.answer("saveCards", request, new SaveCardsResponse())
        if (!this.admin.refusals.has("saveCards")) {
            this.decks.set(request.ruleSet ?? "", (request.cards ?? []).map(card => new CardDraft(card)))
        }
        return answer
    }
}

// A few of planet.v1.Rules' numbers and durations, of bonuses.v1.Rules' under `bonuses.`, of boxes.v1.Rules' under
// `boxes.` and of quizzes.v1.Rules' under `quizzes.`, as GetCards lists them.
const PLACEHOLDER_PATHS = [
    "clicks.tempo", "clicks.burst", "charges.bomb.rings", "charges.shield.held", "bonuses.shield.maxPerBox",
    "boxes.minInterval", "boxes.maxInterval", "quizzes.offerTtl", "quizzes.answerWindow",
]

// The example battle's values, as scoring scores a ×3 battle.
const TRIPLE = {"battle.times": "3", "battle.points.first": "75", "battle.points.second": "54", "battle.points.third": "45"}
