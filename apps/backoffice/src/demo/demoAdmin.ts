import {Duration, PartialMessage, Timestamp} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {
    BanPlayerRequest,
    BanPlayerResponse,
    GetRulesResponse as AntibotGetRulesResponse,
    RuleSet as AntibotRuleSet,
    InspectPlayerRequest,
    InspectPlayerResponse,
    UnbanPlayerRequest,
    UnbanPlayerResponse,
    WatchdogReading,
} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {
    GetMessageRequest,
    GetMessageResponse,
    KeptMessage,
    KeptQuote,
    MuteRequest,
    MuteResponse,
    RemoveMessageRequest,
    RemoveMessageResponse,
} from "../gen/grpc/chat/v1/admin_pb.ts"
import {
    FindPlayersRequest,
    FindPlayersResponse,
    GetRulesResponse,
    Player,
    RevertPlayerRequest,
    RevertPlayerResponse,
    RuleSet,
    TopPlayersRequest,
    TopPlayersResponse,
} from "../gen/grpc/planet/v1/admin_pb.ts"
import {Rules as PlanetRules} from "../gen/grpc/planet/v1/rules_pb.ts"
import {
    CreateVideoUploadRequest,
    CreateVideoUploadResponse,
    KeepVideoRequest,
    KeepVideoResponse,
    Video,
} from "../gen/grpc/clips/v1/admin_pb.ts"
import {AdminGetPlayerRequest, AdminGetPlayerResponse} from "../gen/grpc/player/v1/admin_pb.ts"
import {Author} from "../gen/grpc/player/v1/internal_pb.ts"
import {CountryTiles, Stats} from "../gen/grpc/player/v1/player_pb.ts"
import {
    DecideReportRequest,
    DecideReportResponse,
    ListReportsRequest,
    ListReportsResponse,
    Report,
    ReportDecision,
} from "../gen/grpc/support/v1/admin_pb.ts"
import {ReportReason} from "../gen/grpc/support/v1/support_pb.ts"
import {FakeAdmin} from "../fakeAdmin.ts"

export const OPERATOR = "00000000-0000-4000-8000-00000000c0de"

const NOW = Date.now()
const MINUTE = 60_000
const HOUR = 60 * MINUTE
const LATENCY_MS = 250

const FLAGS = ["fr", "dz", "de", "it", "es", "pt", "us", "gb", "br", "tr", "se", "ie", "pl", "ma", "nl", "ca", "ar", "mx"]
const FIRST = ["Brave", "Swift", "Red", "Silent", "Iron", "Lucky", "Wild", "Dark", "Happy", "Mega", "Tiny", "Royal"]
const SECOND = ["Fox", "Wolf", "Tiger", "Eagle", "Panda", "Shark", "Bear", "Hawk", "Lion", "Otter", "Viper", "Moose"]
const BANTER = [
    "gg", "who is painting algeria??", "france stop", "lol", "come defend portugal", "anyone from brazil?",
    "nice bomb", "why is the map frozen", "help us in europe", "we take spain tonight", "ok", "hahaha",
    "the box gave me shields", "turkey is everywhere", "stop taking our tiles pls", "brb", "sweden rises",
    "how do you get the bomb", "this game is so good", "who wants an alliance", "italy and france together",
]
const ABUSE = ["you all suck", "kys noob", "shut up idiot", "go back to your country", "stupid french dogs"]
const VIDEO_TYPES = ["video/mp4", "video/quicktime", "video/webm"]
const VIDEOS = [
    {
        name: "season-1-final-france", hours: 2, mb: 24.6,
        caption: "France held the map to the last second. Who takes it in Season 2?\n#clickplanet #map #france",
    },
    {name: "season-1-final-algeria", hours: 3, mb: 31.2, caption: "Algeria came back in the last hour.\n#clickplanet #algeria"},
    {name: "season-1-top-10", hours: 26, mb: 8.4, caption: ""},
]

// The same numbers on every boot, so a page looks the same after a reload.
function random(seed: number): () => number {
    let state = seed
    return () => {
        state = (state * 1664525 + 1013904223) % 4294967296
        return state / 4294967296
    }
}

const at = (ms: number) => Timestamp.fromDate(new Date(ms))
const seconds = (n: number) => new Duration({seconds: BigInt(Math.round(n))})

type Account = {id: string, name: string, guest: boolean, flag: string, scope: string, createdAt: number, bot: boolean}

// A fake of every admin method that keeps what the operator does to it, over a few hundred made-up players, for
// `npm run demo`.
export class DemoAdmin extends FakeAdmin {
    accounts: Account[]
    mutedUntil = new Map<string, number>()
    inspections = new Map<string, InspectPlayerResponse>()
    readonly bucket = new Map<string, Blob>()

    constructor() {
        super()
        const roll = random(42)
        const pick = <T, >(list: T[]) => list[Math.floor(roll() * list.length)]
        const hex = (n: number) => Array.from({length: n}, () => Math.floor(roll() * 16).toString(16)).join("")
        const scopes = Array.from({length: 70}, (_, i) => i % 3 === 0
            ? `2a01:e0a:${hex(3)}:${hex(4)}::/64`
            : `${Math.floor(roll() * 200) + 20}.${Math.floor(roll() * 255)}.${Math.floor(roll() * 255)}.${i}`)
        this.accounts = Array.from({length: 120}, (_, i) => {
            const guest = roll() < 0.45
            return {
                id: `${hex(8)}-${hex(4)}-4${hex(3)}-a${hex(3)}-${hex(12)}`,
                name: guest ? `guest_${hex(6)}` : `${pick(FIRST)}${pick(SECOND)}${Math.floor(roll() * 99)}`,
                guest,
                flag: pick(FLAGS),
                // A few households: several accounts behind one address.
                scope: scopes[Math.floor(roll() * scopes.length)],
                createdAt: NOW - Math.floor(roll() * 40 * 24) * HOUR,
                bot: i % 29 === 7,
            }
        })
        // Some accounts never got a name: most likely scripts.
        for (const account of this.accounts.filter(account => account.bot)) account.name = ""

        this.authors = this.accounts.filter(account => account.name !== "")
            .map(account => new Author({accountId: account.id, name: account.name, guest: account.guest}))

        this.players = [
            ...this.accounts.map(account => {
                const takes = account.bot ? 4000 + Math.floor(roll() * 9000) : Math.floor(roll() ** 2 * 3000) + 3
                const activeMinutes = Math.max(1, Math.floor(roll() * 600))
                const tiles = Math.floor(takes * (0.02 + roll() * 0.3))
                const lastAt = NOW - Math.floor(roll() * 70) * HOUR - Math.floor(roll() * 60) * MINUTE
                return new Player({
                    accountId: account.id, scope: account.scope, takes, tiles, flagCountryId: account.flag,
                    takesPerMinute: takes / activeMinutes, tilesPerMinute: tiles / activeMinutes,
                    activeFor: seconds(activeMinutes * 60), firstAt: at(lastAt - activeMinutes * MINUTE), lastAt: at(lastAt),
                })
            }),
            ...scopes.slice(0, 8).map(scope => new Player({
                scope, takes: Math.floor(roll() * 200), tiles: Math.floor(roll() * 20), flagCountryId: pick(FLAGS),
                activeFor: seconds(600), lastAt: at(NOW - Math.floor(roll() * 50) * HOUR),
            })),
        ]
        for (const row of this.players.filter((_, i) => i % 17 === 3)) {
            row.banned = true
            row.bannedUntil = at(NOW + 5 * 24 * HOUR)
        }

        // The chat of the last day and a half, a line every few minutes, now and then an insult.
        const chatty = this.accounts.filter(account => account.name !== "").slice(0, 30)
        let sentAt = NOW - 36 * HOUR
        this.chat = []
        for (let n = 0; sentAt < NOW - 2 * MINUTE; n++) {
            const author = pick(chatty)
            const abusive = roll() < 0.06
            const previous = this.chat[this.chat.length - 1]
            this.chat.push(new KeptMessage({
                messageId: `m-${n}`, accountId: author.id, authorName: author.name, countryId: author.flag,
                text: abusive ? pick(ABUSE) : pick(BANTER), sentAt: at(sentAt),
                replyTo: previous !== undefined && roll() < 0.2
                    ? new KeptQuote({messageId: previous.messageId, authorName: previous.authorName, text: previous.text})
                    : undefined,
            }))
            sentAt += Math.floor(roll() * 6 + 1) * MINUTE
        }

        const nameOf = (id: string) => this.accounts.find(account => account.id === id)?.name ?? ""
        const abusive = this.chat.filter(message => ABUSE.includes(message.text)).reverse()
        const reporters = chatty.slice(10)
        const report = (n: number, part: PartialMessage<Report>) => new Report({
            id: `report-${n}`, reporterName: nameOf(part.reporterAccountId ?? ""),
            reportedName: nameOf(part.reportedAccountId ?? ""), ...part,
        })
        this.reports = [
            ...abusive.slice(0, 5).map((message, n) => report(n, {
                reporterAccountId: reporters[n].id, reportedAccountId: message.accountId,
                reason: ReportReason.CHAT_ABUSE, messageId: message.messageId,
                text: n % 2 === 0 ? "insults everyone in the chat" : "",
                createdAt: at((message.sentAt?.toDate().getTime() ?? NOW) + 3 * MINUTE),
            })),
            report(5, {
                reporterAccountId: reporters[6].id, reportedAccountId: this.accounts[7].id, reason: ReportReason.CHEATING,
                text: "takes 20 tiles a second all night, surely a bot", createdAt: at(NOW - 5 * HOUR),
            }),
            report(6, {
                reporterAccountId: reporters[7].id, reportedAccountId: chatty[3].id, reason: ReportReason.BAD_NAME,
                createdAt: at(NOW - 9 * HOUR),
            }),
            report(7, {
                reporterAccountId: reporters[8].id, reportedAccountId: chatty[4].id, reason: ReportReason.OTHER,
                text: "he says he will hack the game", createdAt: at(NOW - 20 * HOUR),
            }),
            ...abusive.slice(5, 9).map((message, n) => report(8 + n, {
                reporterAccountId: reporters[n].id, reportedAccountId: message.accountId,
                reason: ReportReason.CHAT_ABUSE, messageId: message.messageId,
                createdAt: at((message.sentAt?.toDate().getTime() ?? NOW) + 2 * MINUTE),
                decision: new ReportDecision({
                    outcome: n % 2 === 0 ? 1 : 2, comment: n % 2 === 0 ? "muted a day" : "",
                    decidedAt: at((message.sentAt?.toDate().getTime() ?? NOW) + HOUR),
                }),
            })),
        ].sort((a, b) => Number((b.createdAt?.seconds ?? 0n) - (a.createdAt?.seconds ?? 0n)))

        // The sets the schedule names, in both modules: calm plays now.
        const [planetDefault] = this.rules.sets
        const [antibotDefault] = this.antibotRules.sets
        const variant = (rules: PlanetRules, change: (rules: PlanetRules) => void) => {
            const copy = rules.clone()
            change(copy)
            return copy
        }
        this.rules = new GetRulesResponse({playing: "calm", sets: [
            planetDefault,
            new RuleSet({name: "calm", rules: variant(planetDefault.rules ?? new PlanetRules(), rules => {
                if (rules.clicks !== undefined) rules.clicks.tempo = 0.5
            })}),
            new RuleSet({name: "frozen", rules: variant(planetDefault.rules ?? new PlanetRules(), rules => rules.frozen = true)}),
            new RuleSet({name: "rush", rules: variant(planetDefault.rules ?? new PlanetRules(), rules => {
                if (rules.clicks !== undefined) rules.clicks.tempo = 3
            })}),
        ]})
        this.antibotRules = new AntibotGetRulesResponse({playing: "calm", sets: ["default", "calm", "frozen", "rush"].map(name =>
            new AntibotRuleSet({name, rules: antibotDefault.rules?.clone()}))})

        // Their files are not in the demo: the page says it cannot play them.
        this.videos = VIDEOS.map(({name, caption, hours, mb}, n) => new Video({
            videoId: `demo-video-${n + 1}`, name, caption, contentType: "video/mp4",
            sizeBytes: BigInt(Math.round(mb * 1024 * 1024)), keptAt: at(NOW - hours * HOUR),
            viewUrl: `/demo/${name}.mp4`, downloadUrl: `/demo/${name}.mp4`,
        }))
    }

    override answer<T>(method: string, request: unknown, value: T): Promise<T> {
        return new Promise(resolve => setTimeout(resolve, LATENCY_MS)).then(() => super.answer(method, request, value))
    }

    private refuseLater<T>(method: string, request: unknown, why: string, code: Code): Promise<T> {
        this.asked.push({method, request})
        return new Promise((_, reject) => setTimeout(() => reject(new ConnectError(why, code)), LATENCY_MS))
    }

    // The made-up bucket takes a file in two seconds, a tenth at a time.
    receive(url: string, file: Blob, onSent: (bytes: number) => void): Promise<void> {
        const videoId = url.split("/").pop() ?? ""
        return new Promise(resolve => {
            let tenths = 0
            const timer = setInterval(() => {
                tenths += 1
                onSent(file.size * tenths / 10)
                if (tenths < 10) return
                clearInterval(timer)
                this.bucket.set(videoId, file)
                resolve()
            }, 200)
        })
    }

    override createVideoUpload(request: PartialMessage<CreateVideoUploadRequest>): Promise<CreateVideoUploadResponse> {
        const type = request.contentType ?? ""
        if (!VIDEO_TYPES.includes(type)) {
            return this.refuseLater("createVideoUpload", request,
                `"${type}" is not video/mp4, video/quicktime or video/webm`, Code.InvalidArgument)
        }
        return super.createVideoUpload(request)
    }

    override keepVideo(request: PartialMessage<KeepVideoRequest>): Promise<KeepVideoResponse> {
        const file = this.bucket.get(request.videoId ?? "")
        if (file === undefined) {
            return this.refuseLater("keepVideo", request, "the file is not in the bucket", Code.FailedPrecondition)
        }
        const link = URL.createObjectURL(file)
        this.videos = [new Video({
            videoId: request.videoId, name: request.name, caption: request.caption, contentType: file.type,
            sizeBytes: BigInt(file.size), keptAt: at(Date.now()), viewUrl: link, downloadUrl: link,
        }), ...this.videos]
        return this.answer("keepVideo", request, new KeepVideoResponse())
    }

    override listReports(request: PartialMessage<ListReportsRequest>): Promise<ListReportsResponse> {
        const reports = this.reports.filter(report => !request.openOnly || report.decision === undefined)
        return this.answer("listReports", request, new ListReportsResponse({reports}))
    }

    override decideReport(request: PartialMessage<DecideReportRequest>): Promise<DecideReportResponse> {
        const decidedAt = at(Date.now())
        this.reports = this.reports.map(report => report.id !== request.reportId ? report : new Report({
            ...report, decision: new ReportDecision({outcome: request.outcome, comment: request.comment, decidedAt}),
        }))
        return this.answer("decideReport", request, new DecideReportResponse({decidedAt}))
    }

    override getMessage(request: PartialMessage<GetMessageRequest>): Promise<GetMessageResponse> {
        const kept = this.chat.find(message => message.messageId === request.messageId)
        if (kept === undefined) return this.refuseLater("getMessage", request, "no message kept under this id", Code.NotFound)
        return this.answer("getMessage", request, new GetMessageResponse({
            messageId: kept.messageId, accountId: kept.accountId, text: kept.text, sentAt: kept.sentAt,
            countryId: kept.countryId, removedAt: kept.removedAt,
        }))
    }

    override removeMessage(request: PartialMessage<RemoveMessageRequest>): Promise<RemoveMessageResponse> {
        const removedAt = at(Date.now())
        const kept = this.chat.find(message => message.messageId === request.messageId)
        if (kept !== undefined) kept.removedAt = removedAt
        return this.answer("removeMessage", request, new RemoveMessageResponse({removedAt}))
    }

    override mute(request: PartialMessage<MuteRequest>): Promise<MuteResponse> {
        const until = Date.now() + Number(request.duration?.seconds ?? 3600n) * 1000
        this.mutedUntil.set(request.accountId ?? "", until)
        return this.answer("mute", request, new MuteResponse({mutedUntil: at(until)}))
    }

    override topPlayers(request: PartialMessage<TopPlayersRequest>): Promise<TopPlayersResponse> {
        const sorted = [...this.players].sort((a, b) => b.takes - a.takes || b.tiles - a.tiles)
        return this.answer("topPlayers", request,
            new TopPlayersResponse({players: sorted.slice(0, request.limit || 20), total: sorted.length}))
    }

    override findPlayers(request: PartialMessage<FindPlayersRequest>): Promise<FindPlayersResponse> {
        const found = this.players.filter(row => row.flagCountryId === request.flagCountryId)
            .sort((a, b) => Number((b.lastAt?.seconds ?? 0n) - (a.lastAt?.seconds ?? 0n)))
        return this.answer("findPlayers", request,
            new FindPlayersResponse({players: found.slice(0, request.limit || 20), total: found.length}))
    }

    override revertPlayer(request: PartialMessage<RevertPlayerRequest>): Promise<RevertPlayerResponse> {
        const rows = this.rowsOf(request)
        const held = rows.reduce((sum, row) => sum + row.tiles, 0)
        const touched = rows.reduce((sum, row) => sum + row.takes, 0)
        if (request.dryRun !== true) for (const row of rows) row.tiles = 0
        return this.answer("revertPlayer", request,
            new RevertPlayerResponse({touched, held, restored: request.dryRun === true ? 0 : held}))
    }

    override getPlayer(request: PartialMessage<AdminGetPlayerRequest>): Promise<AdminGetPlayerResponse> {
        const account = this.accounts.find(each => each.id === request.accountId)
        if (account === undefined) return this.refuseLater("getPlayer", request, "no such account", Code.NotFound)
        const row = this.players.find(each => each.accountId === account.id)
        const taken = BigInt((row?.takes ?? 0) * 3)
        return this.answer("getPlayer", request, new AdminGetPlayerResponse({
            name: account.name, guest: account.guest, linked: !account.guest, createdAtUnixMs: BigInt(account.createdAt),
            messagesSent: BigInt(this.chat.filter(message => message.accountId === account.id).length),
            stats: new Stats({tilesTaken: taken, streakCurrent: 2, streakBest: 9}),
            playsFor: [
                new CountryTiles({countryId: account.flag, tiles: taken * 9n / 10n}),
                new CountryTiles({countryId: "gb-sct", tiles: taken / 10n}),
            ],
            playsAgainst: FLAGS.filter(flag => flag !== account.flag).slice(0, 8)
                .map((countryId, i) => new CountryTiles({countryId, tiles: taken / BigInt(i + 3)})),
        }))
    }

    private rowsOf(caller: {accountId?: string, scope?: string}): Player[] {
        return this.players.filter(row => caller.accountId ? row.accountId === caller.accountId : row.scope === caller.scope)
    }

    private inspectionOf(caller: {accountId?: string, scope?: string}): InspectPlayerResponse {
        const key = caller.accountId || caller.scope || ""
        const kept = this.inspections.get(key)
        if (kept !== undefined) return kept
        const row = this.rowsOf(caller)[0]
        const bot = this.accounts.find(account => account.id === caller.accountId)?.bot ?? false
        const fresh = new InspectPlayerResponse({
            accountId: caller.accountId ?? "", scope: row?.scope ?? caller.scope ?? "",
            tracked: bot, flags: bot ? 14 : 0, banned: row?.banned ?? false, bannedUntil: row?.bannedUntil,
            offence: row?.banned ? 1 : 0, suspects: bot ? 3 : 0, minSuspects: 2, guilty: bot,
            clicks: row?.takes ?? 0, activeFor: row?.activeFor, longestGap: seconds(bot ? 3 : 420),
            lastClickAt: row?.lastAt, topCountry: row?.flagCountryId ?? "", topCountryClicks: row?.takes ?? 0,
            readings: bot
                ? [
                    new WatchdogReading({watchdog: "metronome", level: "certain", evidence: "1,240 clicks 0.98 s apart, spread 0.02 s", at: row?.lastAt}),
                    new WatchdogReading({watchdog: "sequencer", level: "suspect", evidence: "62 steps along a row, 81% of its takes", at: row?.lastAt}),
                    new WatchdogReading({watchdog: "hopper", level: "suspect", evidence: "14 hops over 120° in 1 s", at: row?.lastAt}),
                ]
                : [new WatchdogReading({watchdog: "metronome", level: "clear", at: row?.lastAt})],
        })
        this.inspections.set(key, fresh)
        return fresh
    }

    readonly demoAntibot = {
        inspectPlayer: (request: PartialMessage<InspectPlayerRequest>) =>
            this.answer("inspectPlayer", request, this.inspectionOf(request)),
        banPlayer: (request: PartialMessage<BanPlayerRequest>) => {
            const inspection = this.inspectionOf(request)
            const offence = inspection.offence + 1
            const days = [1, 7, 1095][Math.min(offence, 3) - 1]
            const until = at(Date.now() + (request.duration ? Number(request.duration.seconds) * 1000 : days * 24 * HOUR))
            inspection.banned = true
            inspection.offence = offence
            inspection.bannedUntil = until
            for (const row of this.rowsOf(request)) {
                row.banned = true
                row.bannedUntil = until
            }
            return this.answer("banPlayer", request, new BanPlayerResponse({bannedUntil: until, offence, enforced: true}))
        },
        unbanPlayer: (request: PartialMessage<UnbanPlayerRequest>) => {
            const inspection = this.inspectionOf(request)
            if (inspection.offence === 0) return this.refuseLater<UnbanPlayerResponse>("unbanPlayer", request, "no ban", Code.NotFound)
            const offence = inspection.offence
            const until = inspection.bannedUntil
            inspection.banned = false
            inspection.offence = offence - 1
            inspection.bannedUntil = undefined
            const pardonedUntil = at(Date.now() + 24 * HOUR)
            inspection.pardonedUntil = pardonedUntil
            for (const row of this.rowsOf(request)) {
                row.banned = false
                row.bannedUntil = undefined
            }
            return this.answer("unbanPlayer", request, new UnbanPlayerResponse({offence, bannedUntil: until, pardonedUntil}))
        },
        getRules: this.antibot.getRules.bind(this.antibot),
        saveRuleSet: this.antibot.saveRuleSet.bind(this.antibot),
    }
}
