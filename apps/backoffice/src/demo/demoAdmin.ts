import {textBody, wordsIn} from "../chat/said.ts"
import {labOf} from "./demoLab.ts"
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
    WarnPlayerRequest,
    WarnPlayerResponse,
    WatchdogReading,
} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {
    GetMessageRequest,
    GetMessageResponse,
    GetMuteRequest,
    GetMuteResponse,
    KeptMessage,
    KeptQuote,
    ListModerationRequest,
    ListModerationResponse,
    ModerationAction,
    MuteAction,
    MuteRequest,
    MuteResponse,
    RemovalAction,
    RemoveMessageRequest,
    RemoveMessageResponse,
    UnmuteAction,
    UnmuteRequest,
    UnmuteResponse,
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
import {GetTerritoriesResponse} from "../gen/grpc/planet/v1/internal_pb.ts"
import {Rules as AntibotRules} from "../gen/grpc/antibot/v1/rules_pb.ts"
import {GetRulesResponse as BonusesGetRulesResponse, RuleSet as BonusesRuleSet} from "../gen/grpc/bonuses/v1/admin_pb.ts"
import {Rules as BonusesRules} from "../gen/grpc/bonuses/v1/rules_pb.ts"
import {GetRulesResponse as BoxesGetRulesResponse, RuleSet as BoxesRuleSet} from "../gen/grpc/boxes/v1/admin_pb.ts"
import {Rules as BoxesRules} from "../gen/grpc/boxes/v1/rules_pb.ts"
import {GetRulesResponse as QuizzesGetRulesResponse, RuleSet as QuizzesRuleSet} from "../gen/grpc/quizzes/v1/admin_pb.ts"
import {Rules as QuizzesRules} from "../gen/grpc/quizzes/v1/rules_pb.ts"
import {MuteKind} from "../gen/grpc/chat/v1/chat_pb.ts"
import {
    GetPlayerLinkRequest,
    GetPlayerLinkResponse,
    LinkMembersRequest,
    LinkMembersResponse,
    UnlinkPlayerRequest,
    UnlinkPlayerResponse,
} from "../gen/grpc/alerts/v1/admin_pb.ts"
import {Rules as PlanetRules} from "../gen/grpc/planet/v1/rules_pb.ts"
import {
    CreateVideoUploadRequest,
    CreateVideoUploadResponse,
    KeepVideoRequest,
    KeepVideoResponse,
    Video,
} from "../gen/grpc/clips/v1/admin_pb.ts"
import {
    AdminGetPlayerRequest,
    AdminGetPlayerResponse,
    FormerName,
    ListRenamesRequest,
    ListRenamesResponse,
    Rename,
    RenamePlayerRequest,
    RenamePlayerResponse,
} from "../gen/grpc/player/v1/admin_pb.ts"
import {Player as Identity} from "../gen/grpc/player/v1/player_pb.ts"
import {AdminGetProgressRequest, AdminGetProgressResponse} from "../gen/grpc/progression/v1/admin_pb.ts"
import {CountryTiles, Stats} from "../gen/grpc/progression/v1/progression_pb.ts"
import {
    DecideReportRequest,
    DecideReportResponse,
    Letter,
    LetterMail,
    ListReportsRequest,
    ListReportsResponse,
    Note,
    NoteAnswer,
    NoteKind,
    NoteReply,
    Report,
    ReportDecision,
} from "../gen/grpc/support/v1/admin_pb.ts"
import {PageContext, ReportReason} from "../gen/grpc/support/v1/support_pb.ts"
import {FakeAdmin, keptSubmission, soldFlag} from "../fakeAdmin.ts"
import {
    CreateFlagUploadRequest,
    CreateFlagUploadResponse,
    KeptSubmission,
    OpenFlagRequest,
    OpenFlagResponse,
} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Flag, Picture, Rejection, RejectionReason, Sold, SubmissionState} from "../gen/grpc/flags/v1/flags_pb.ts"
import {Graffiti, GraffitiKind} from "../gen/grpc/vandalsquad/v1/admin_pb.ts"
import {
    Background as PromoBackground,
    CardDraft,
    CreateBackgroundUploadRequest,
    CreateBackgroundUploadResponse,
    KeepBackgroundRequest,
    KeepBackgroundResponse,
    RemoveBackgroundRequest,
    RemoveBackgroundResponse,
} from "../gen/grpc/promo/v1/admin_pb.ts"

export const OPERATOR = "00000000-0000-4000-8000-00000000c0de"

const NOW = Date.now()
const MINUTE = 60_000
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR
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

const FORMER_NAMES = ["BraveFox42", "QuietOwl17", "Nova_8", "Tiny Comet", "RedPanda61", "Kartograf", "SilentElk9"]

// A fake of every admin method that keeps what the operator does to it, over a few hundred made-up players, for
// `npm run demo`.
export class DemoAdmin extends FakeAdmin {
    accounts: Account[]
    // The names each account's player left, newest first: the operator's renames are added on read, as the server does.
    formerNames = new Map<string, FormerName[]>()
    // By account and kind: a mute of the GIFs holds apart from one of the whole chat.
    mutedUntil = new Map<string, number>()
    inspections = new Map<string, InspectPlayerResponse>()
    readonly bucket = new Map<string, Blob>()

    constructor() {
        super()
        const lab = labOf(NOW, random(1120))
        this.rowPages = [lab.rows]
        this.arrivals = lab.arrivals
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
        // A few play for a flag opened since.
        this.accounts.filter((_, i) => i % 13 === 6).forEach(account => account.flag = BAVARIA)
        // Some accounts never got a name: most likely scripts.
        for (const account of this.accounts.filter(account => account.bot)) account.name = ""
        // Some players changed their name: up to three, a little over a week apart. Picked by index, so every other
        // made-up value stays as it was.
        this.accounts.forEach((account, i) => {
            if (account.guest || account.name === "") return
            this.formerNames.set(account.id, Array.from({length: i % 4}, (_, n) => new FormerName({
                name: FORMER_NAMES[(i + n * 3) % FORMER_NAMES.length],
                untilUnixMs: BigInt(NOW - (2 + n * 9) * 24 * HOUR),
            })))
        })

        this.identities = this.accounts.filter(account => account.name !== "")
            .map(account => new Identity({accountId: account.id, name: account.name, guest: account.guest}))

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

        // Four drawings the vandal squad flagged, from its own tests' samples: three swastikas, two of them by one
        // player, and one the operator judged none, which a person must clear by eye.
        const painters = this.accounts.filter(account => account.name !== "")
        this.graffiti = DEMO_GRAFFITI.map(({tiles, flag, hoursAgo, painter, place}) => new Graffiti({
            board: "earth-1", accountId: painters[painter].id, countryId: flag, kind: GraffitiKind.SWASTIKA, tileIds: tiles,
            flaggedAt: at(NOW - hoursAgo * HOUR), seenAt: at(NOW - hoursAgo * HOUR + 40 * MINUTE),
            link: `https://clickplanet.lol/play?lat=${place[0]}&lon=${place[1]}&zoom=20.0`,
        }))

        // The chat of the last day and a half, a line every few minutes, now and then an insult, one in five in the
        // author's flag room.
        const chatty = this.accounts.filter(account => account.name !== "").slice(0, 30)
        let sentAt = NOW - 36 * HOUR
        this.chat = []
        for (let n = 0; sentAt < NOW - 2 * MINUTE; n++) {
            const author = pick(chatty)
            const abusive = roll() < 0.06
            const previous = this.chat[this.chat.length - 1]
            this.chat.push(new KeptMessage({
                messageId: `m-${n}`, accountId: author.id, authorName: author.name, countryId: author.flag,
                flagRoom: n % 5 === 4 ? author.flag : "",
                body: textBody(abusive ? pick(ABUSE) : pick(BANTER)), sentAt: at(sentAt),
                replyTo: previous !== undefined && roll() < 0.2
                    ? new KeptQuote({messageId: previous.messageId, authorName: previous.authorName, body: previous.body})
                    : undefined,
            }))
            sentAt += Math.floor(roll() * 6 + 1) * MINUTE
        }

        const nameOf = (id: string) => this.accounts.find(account => account.id === id)?.name ?? ""
        const abusive = this.chat.filter(message => ABUSE.includes(wordsIn(message.body))).reverse()

        // What Claude did on its own runs: the oldest insults removed, a second one muted 15 minutes.
        const removed = abusive.slice(-2)
        for (const message of removed) message.removedAt = at((message.sentAt?.toDate().getTime() ?? NOW) + 40 * MINUTE)
        this.moderation = removed.flatMap((message, n) => {
            const actedAt = message.removedAt ?? at(NOW)
            const removal = new ModerationAction({
                actionId: String(2 * n + 1), accountId: message.accountId, actedAt, client: "Claude",
                operatorAccountId: OPERATOR, reason: n === 0 ? "offence 1 (chat): an insult aimed at a player" : "spam, no offence",
                action: {case: "removal", value: new RemovalAction({
                    messageId: message.messageId, body: {case: "text", value: wordsIn(message.body)},
                })},
            })
            if (n !== 0) return [removal]
            return [new ModerationAction({
                actionId: String(2 * n + 2), accountId: message.accountId, actedAt, client: "Claude", operatorAccountId: OPERATOR,
                reason: "offence 2 (chat): removed, muted 15 minutes",
                action: {case: "mute", value: new MuteAction({kind: MuteKind.CHAT, duration: new Duration({seconds: 900n})})},
            }), removal]
        }).reverse()
        this.renames = []
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

        // Two notes waiting for a first answer, one the player replied to, one a parent replied to, one answered, one
        // closed, and one sent before the notes kept an address.
        const writers = this.accounts.filter(account => !account.guest && account.name !== "")
        const addressOf = (n: number) => `${writers[n].name.toLowerCase()}@example.com`
        const note = (n: number, part: PartialMessage<Note>) => new Note({
            id: `note-${n}`, accountId: writers[n].id, replyAddress: addressOf(n), ...part,
        })
        this.notes = [
            note(0, {
                kind: NoteKind.BUG, text: "the globe goes black after a bomb on my phone", submittedAt: at(NOW - 40 * MINUTE),
                userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_1 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148",
                context: new PageContext({
                    build: "2026-11-22", viewportWidth: 390, viewportHeight: 844, pixelRatio: 3, webglVersion: "WebGL 2.0",
                    maxTextureSize: 4096, cores: 6,
                }),
            }),
            note(1, {kind: NoteKind.SUGGESTION, text: "let us pick a second flag for the battles", submittedAt: at(NOW - 3 * HOUR)}),
            note(2, {
                kind: NoteKind.BUG, text: "my clicks stop after 10 minutes", submittedAt: at(NOW - 2 * DAY),
                answers: [new NoteAnswer({text: "Thanks! Which browser do you play on?", sentAt: at(NOW - 2 * DAY + 5 * HOUR)})],
                replies: [new NoteReply({text: "chrome on windows", from: addressOf(2), receivedAt: at(NOW - 6 * HOUR)})],
            }),
            note(3, {
                kind: NoteKind.SUGGESTION, text: "a map of the old seasons", submittedAt: at(NOW - 3 * DAY),
                answers: [new NoteAnswer({text: "Good idea, it is on the list.", sentAt: at(NOW - 3 * DAY + 2 * HOUR)})],
                replies: [new NoteReply({
                    text: "This is his mother. He plays too much, please tell him to sleep.", from: "parent@example.org",
                    receivedAt: at(NOW - 26 * HOUR),
                })],
            }),
            note(4, {
                kind: NoteKind.SUGGESTION, text: "dark mode please", submittedAt: at(NOW - 4 * DAY),
                answers: [new NoteAnswer({text: "It follows your phone's setting now.", sentAt: at(NOW - 4 * DAY + 9 * HOUR)})],
            }),
            note(5, {kind: NoteKind.BUG, text: "asdfgh", submittedAt: at(NOW - 5 * DAY), closedAt: at(NOW - 5 * DAY + 2 * HOUR)}),
            note(6, {kind: NoteKind.SUGGESTION, text: "more countries in Africa", submittedAt: at(NOW - 40 * DAY), replyAddress: ""}),
        ]

        // The sets the schedule names, in each module: calm plays now. Each holds what it changes from default.
        const [planetDefault] = this.rules.sets
        const [antibotDefault] = this.antibotRules.sets
        const [bonusesDefault] = this.bonusRules.sets
        const [boxesDefault] = this.boxRules.sets
        const [quizzesDefault] = this.quizRules.sets
        this.rules = new GetRulesResponse({playing: "calm", sets: [
            planetDefault,
            new RuleSet({name: "calm", rules: PlanetRules.fromJson({clicks: {tempo: 0.5}})}),
            new RuleSet({name: "frozen", rules: PlanetRules.fromJson({frozen: true})}),
            new RuleSet({name: "rush", rules: PlanetRules.fromJson({
                clicks: {tempo: 3}, charges: {bomb: {held: 3}}, allegianceLocked: true,
            })}),
        ]})
        this.flags.listed = [...this.flags.listed.filter(flag => flag.sold === undefined), ...DEMO_FLAGS]
        // Seven flags players submitted: four waiting (Catalonia, fine; Tyrol, which the game had; a photo of Lake Garda;
        // a new name and picture for Bavaria), Sardinia approved and waiting to be opened, Brittany rejected, and Cornwall,
        // opened from its submission.
        const founders = this.accounts.filter(account => account.name !== "" && !account.guest)
        this.flags.submissions = DEMO_SUBMISSIONS.map(({id, name, pin, hoursAgo, decidedHoursAgo, picture, part}, n) => keptSubmission(id, name, pin, {
            accountId: founders[(n * 7 + 3) % founders.length].id,
            submittedAt: at(NOW - hoursAgo * HOUR),
            decidedAt: at(NOW - (decidedHoursAgo ?? hoursAgo) * HOUR),
            picture: new Picture({url: picture, width: 768, height: 512}),
            ...part,
        }))
        // Every founder holds a verified address but Lake Garda's; Tyrol's replied to a letter the operator wrote.
        const addressOfAccount = (accountId: string) =>
            `${(this.accounts.find(account => account.id === accountId)?.name ?? "player").toLowerCase()}@example.com`
        this.addresses = new Map(this.flags.submissions.filter(each => each.id !== "demo-sub-garda")
            .map(each => [each.accountId, addressOfAccount(each.accountId)]))
        const tyrol = this.flags.submissions.find(each => each.id === "demo-sub-tyrol")?.accountId ?? ""
        this.letters = [new Letter({
            id: "demo-letter-1", accountId: tyrol, address: addressOfAccount(tyrol), subject: "Your flag Tyrol",
            mails: [new LetterMail({
                text: "Hi! Tyrol was a sold flag already, closed in October. Do you mean the Austrian state alone, or the whole region with South Tyrol and Trentino?",
                sentAt: at(NOW - 3 * HOUR),
            })],
            replies: [new NoteReply({
                text: "the whole region!! south tyrol and trentino too, my grandparents are from bolzano",
                from: addressOfAccount(tyrol), receivedAt: at(NOW - 40 * MINUTE),
            })],
        })]
        // The board the world plays since 2026-11-18: the submissions' capitals are drawn on it.
        this.territories = new GetTerritoriesResponse({...this.territories, board: "earth-2", tiles: 1004682})
        this.promo.backgrounds = DEMO_BACKGROUNDS
        this.promo.decks.set("rush", [
            new CardDraft({backgroundId: "demo-war", headline: "Clicks refill {clicks.tempo}× faster", line: "Every country fights at once."}),
            new CardDraft({
                backgroundId: "demo-points", headline: "×{battle.times} points",
                line: "1st place wins {battle.points.first} points, 2nd {battle.points.second}, 3rd {battle.points.third}.",
            }),
            new CardDraft({backgroundId: "demo-rally", headline: "Bring your friends", line: "Your flag needs every click."}),
            new CardDraft({
                backgroundId: "demo-war", headline: "A box every {boxes.minInterval}",
                line: "Bombs blast {charges.bomb.rings} rings wide.",
            }),
        ])
        this.antibotRules = new AntibotGetRulesResponse({playing: "calm", sets: [
            antibotDefault,
            ...["calm", "frozen"].map(name => new AntibotRuleSet({name})),
            new AntibotRuleSet({name: "rush", rules: AntibotRules.fromJson({cohort: {certainMembers: 50}})}),
        ]})
        this.boxRules = new BoxesGetRulesResponse({playing: "calm", sets: [
            boxesDefault,
            new BoxesRuleSet({name: "calm", rules: BoxesRules.fromJson({maxBoxesPerHour: 6})}),
            new BoxesRuleSet({name: "frozen", rules: BoxesRules.fromJson({frozen: true})}),
            new BoxesRuleSet({name: "rush", rules: BoxesRules.fromJson({
                minInterval: "60s", maxInterval: "60s", maxBoxesPerHour: 30,
            })}),
        ]})
        this.bonusRules = new BonusesGetRulesResponse({playing: "calm", sets: [
            bonusesDefault,
            new BonusesRuleSet({name: "calm", rules: BonusesRules.fromJson({kinds: {bomb: 0}})}),
            new BonusesRuleSet({name: "frozen"}),
            new BonusesRuleSet({name: "rush", rules: BonusesRules.fromJson({kinds: {bomb: 3}})}),
        ]})
        this.quizRules = new QuizzesGetRulesResponse({playing: "calm", sets: [
            quizzesDefault,
            new QuizzesRuleSet({name: "calm", rules: QuizzesRules.fromJson({enabled: false})}),
            new QuizzesRuleSet({name: "frozen", rules: QuizzesRules.fromJson({frozen: true})}),
            new QuizzesRuleSet({name: "rush", rules: QuizzesRules.fromJson({
                minInterval: "120s", maxInterval: "180s", maxRightAnswersPerHour: 20,
            })}),
        ]})

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

    protected override now(): Date {
        return new Date()
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
            messageId: kept.messageId, accountId: kept.accountId, body: kept.body, sentAt: kept.sentAt,
            countryId: kept.countryId, removedAt: kept.removedAt, flagRoom: kept.flagRoom,
        }))
    }

    // What the operator does on this page, kept as the backend keeps it: newest first, by the operator.
    private acted(accountId: string, action: ModerationAction["action"], reason: string | undefined) {
        this.moderation = [new ModerationAction({
            actionId: String(this.moderation.length + 1), accountId, actedAt: at(Date.now()), action,
            reason: reason ?? "", operatorAccountId: OPERATOR,
        }), ...this.moderation]
    }

    override listModeration(request: PartialMessage<ListModerationRequest>): Promise<ListModerationResponse> {
        const actions = this.moderation.filter(action => !request.accountId || action.accountId === request.accountId)
        return this.answer("listModeration", request, new ListModerationResponse({actions}))
    }

    override listRenames(request: PartialMessage<ListRenamesRequest>): Promise<ListRenamesResponse> {
        const renames = this.renames.filter(rename => !request.accountId || rename.accountId === request.accountId)
        return this.answer("listRenames", request, new ListRenamesResponse({renames}))
    }

    override removeMessage(request: PartialMessage<RemoveMessageRequest>): Promise<RemoveMessageResponse> {
        const removedAt = at(Date.now())
        const kept = this.chat.find(message => message.messageId === request.messageId)
        if (kept !== undefined) {
            if (kept.removedAt === undefined) {
                this.acted(kept.accountId, {case: "removal", value: new RemovalAction({
                    messageId: kept.messageId,
                    body: kept.body.case === "gif" ? {case: "gif", value: kept.body.value} : {case: "text", value: kept.body.value ?? ""},
                })}, request.reason)
            }
            kept.removedAt = removedAt
        }
        return this.answer("removeMessage", request, new RemoveMessageResponse({removedAt}))
    }

    override mute(request: PartialMessage<MuteRequest>): Promise<MuteResponse> {
        const until = Date.now() + Number(request.duration?.seconds ?? 3600n) * 1000
        this.mutedUntil.set(muteKeyOf(request), until)
        this.acted(request.accountId ?? "", {case: "mute", value: new MuteAction({
            kind: request.kind ?? MuteKind.CHAT, duration: new Duration({seconds: request.duration?.seconds ?? 3600n}),
        })}, request.reason)
        return this.answer("mute", request, new MuteResponse({mutedUntil: at(until)}))
    }

    override unmute(request: PartialMessage<UnmuteRequest>): Promise<UnmuteResponse> {
        const until = this.mutedUntil.get(muteKeyOf(request))
        if (until === undefined || until <= Date.now()) {
            return this.refuseLater("unmute", request, "no mute applies to this caller", Code.NotFound)
        }
        this.mutedUntil.delete(muteKeyOf(request))
        this.acted(request.accountId ?? "", {case: "unmute", value: new UnmuteAction({kind: request.kind ?? MuteKind.CHAT})},
            request.reason)
        return this.answer("unmute", request, new UnmuteResponse({mutedUntil: at(until)}))
    }

    override getMute(request: PartialMessage<GetMuteRequest>): Promise<GetMuteResponse> {
        const until = this.mutedUntil.get(muteKeyOf(request))
        const answer = until === undefined || until <= Date.now()
            ? new GetMuteResponse()
            : new GetMuteResponse({mutedUntil: at(until), mutedAccountId: request.accountId})
        return this.answer("getMute", request, answer)
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
        if (account === undefined || account.name === "") {
            return this.refuseLater("getPlayer", request, "no player for this account", Code.NotFound)
        }
        const replaced = this.renames.filter(rename => rename.accountId === account.id).map(rename => new FormerName({
            name: rename.oldName, untilUnixMs: BigInt(rename.renamedAt?.toDate().getTime() ?? NOW), renamedByOperator: true,
        }))
        const formerNames = [...replaced, ...this.formerNames.get(account.id) ?? []]
            .sort((a, b) => Number(b.untilUnixMs - a.untilUnixMs))
        return this.answer("getPlayer", request, new AdminGetPlayerResponse({
            name: account.name, guest: account.guest, linked: !account.guest, createdAtUnixMs: BigInt(account.createdAt),
            formerNames,
        }))
    }

    // Zeros for an account it keeps nothing about, as progression answers.
    private progressOf(accountId: string): AdminGetProgressResponse {
        const account = this.accounts.find(each => each.id === accountId)
        if (account === undefined) return new AdminGetProgressResponse()
        const row = this.players.find(each => each.accountId === account.id)
        const taken = BigInt((row?.takes ?? 0) * 3)
        return new AdminGetProgressResponse({
            messagesSent: BigInt(this.chat.filter(message => message.accountId === account.id).length),
            stats: new Stats({tilesTaken: taken, streakCurrent: 2, streakBest: 9}),
            playsFor: [
                new CountryTiles({countryId: account.flag, tiles: taken * 9n / 10n}),
                new CountryTiles({countryId: "gb-sct", tiles: taken / 10n}),
            ],
            playsAgainst: FLAGS.filter(flag => flag !== account.flag).slice(0, 8)
                .map((countryId, i) => new CountryTiles({countryId, tiles: taken / BigInt(i + 3)})),
        })
    }

    override renamePlayer(request: PartialMessage<RenamePlayerRequest>): Promise<RenamePlayerResponse> {
        const account = this.accounts.find(each => each.id === request.accountId)
        if (account === undefined || account.guest || account.name === "") {
            return this.refuseLater("renamePlayer", request, "the account has no profile", Code.NotFound)
        }
        const old = account.name
        account.name = (request.name ?? "").trim()
        this.renames = [new Rename({
            renameId: String(this.renames.length + 1), accountId: account.id, renamedAt: at(Date.now()), oldName: old,
            newName: account.name, reason: request.reason ?? "", operatorAccountId: OPERATOR,
        }), ...this.renames]
        this.identities = this.identities.map(identity => identity.accountId === account.id
            ? new Identity({...identity, name: account.name})
            : identity)
        return this.answer("renamePlayer", request, new RenamePlayerResponse({name: account.name}))
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

    // Every account is linked to Discord until the page unlinks it.
    private readonly unlinkedFromDiscord = new Set<string>()
    // Players signed in with Discord before it joined the server, and in it by the invite: the first LinkMembers links them.
    private discordMembersWaiting = 4

    readonly demoAlerts = {
        getPlayerLink: (request: PartialMessage<GetPlayerLinkRequest>) => this.answer("getPlayerLink", request,
            new GetPlayerLinkResponse({
                link: this.unlinkedFromDiscord.has(request.accountId ?? "") ? undefined : this.discordLink,
            })),
        unlinkPlayer: (request: PartialMessage<UnlinkPlayerRequest>) => {
            const accountId = request.accountId ?? ""
            const unlinked = !this.unlinkedFromDiscord.has(accountId)
            this.unlinkedFromDiscord.add(accountId)
            return this.answer("unlinkPlayer", request, new UnlinkPlayerResponse({unlinked}))
        },
        // An account the page unlinked stayed in the server, so a run links it again; 9 players never joined it. Each
        // one linked now is given its nickname too.
        linkMembers: (request: PartialMessage<LinkMembersRequest>) => {
            const linked = this.discordMembersWaiting + this.unlinkedFromDiscord.size
            this.discordMembersWaiting = 0
            this.unlinkedFromDiscord.clear()
            return this.answer("linkMembers", request, new LinkMembersResponse({linked, named: linked, outside: 9}))
        },
    }

    // promo.v1, as the server answers it: an image only, the file in the made-up bucket, and no background a card shows
    // removed.
    readonly demoPromo = {
        createBackgroundUpload: (request: PartialMessage<CreateBackgroundUploadRequest>) => {
            const type = request.contentType ?? ""
            if (!IMAGE_TYPES.includes(type)) {
                return this.refuseLater<CreateBackgroundUploadResponse>("createBackgroundUpload", request, `"${type}" is not a JPEG, a PNG or a WebP`, Code.InvalidArgument)
            }
            return this.promo.createBackgroundUpload(request)
        },
        keepBackground: async (request: PartialMessage<KeepBackgroundRequest>) => {
            const file = this.bucket.get(request.backgroundId ?? "")
            if (file === undefined) {
                return this.refuseLater<KeepBackgroundResponse>("keepBackground", request, "the file is not in the bucket",
                    Code.FailedPrecondition)
            }
            const answer = await this.promo.keepBackground(request)
            this.promo.backgrounds = this.promo.backgrounds.map(background => background.backgroundId === request.backgroundId
                ? new PromoBackground({...background, url: URL.createObjectURL(file)})
                : background)
            return answer
        },
        listBackgrounds: this.promo.listBackgrounds.bind(this.promo),
        removeBackground: (request: PartialMessage<RemoveBackgroundRequest>) => {
            const sets = [...this.promo.decks].filter(([, cards]) => cards.some(card => card.backgroundId === request.backgroundId))
            if (sets.length > 0) {
                return this.refuseLater<RemoveBackgroundResponse>("removeBackground", request,
                    `a card shows this background: ${request.backgroundId}, on the cards of ${sets.map(([set]) => set).join(", ")}`,
                    Code.FailedPrecondition)
            }
            return this.promo.removeBackground(request)
        },
        getCards: this.promo.getCards.bind(this.promo),
        saveCards: this.promo.saveCards.bind(this.promo),
    }

    // flags.v1, as the server answers it: a PNG, a JPEG or a WebP of 5 MB at most, the file in the made-up bucket, a
    // name of 1 to 40 characters no other flag has and an end to come; the picture shown from the page's memory. A rename
    // as FakeFlags keeps it.
    readonly demoFlags = {
        listFlags: this.flags.listFlags.bind(this.flags),
        createFlagUpload: (request: PartialMessage<CreateFlagUploadRequest>) => {
            const type = request.contentType ?? ""
            if (!IMAGE_TYPES.includes(type)) {
                return this.refuseLater<CreateFlagUploadResponse>("createFlagUpload", request, `"${type}" is not a PNG, a JPEG or a WebP`, Code.InvalidArgument)
            }
            if ((request.sizeBytes ?? 0n) > BigInt(FLAG_PICTURE_BYTES)) {
                return this.refuseLater<CreateFlagUploadResponse>("createFlagUpload", request, "a picture of 5 MB at most", Code.InvalidArgument)
            }
            return this.flags.createFlagUpload(request)
        },
        openFlag: async (request: PartialMessage<OpenFlagRequest>) => {
            const file = this.bucket.get(request.uploadId ?? "")
            const name = (request.name ?? "").trim()
            const why = file === undefined ? "no picture uploaded under this id"
                : name === "" || [...name].length > 40 ? "a name of 1 to 40 characters"
                    : new Timestamp(request.endsAt).toDate().getTime() <= Date.now() ? "an end to come" : undefined
            if (file === undefined || why !== undefined) {
                return this.refuseLater<OpenFlagResponse>("openFlag", request, why ?? "", Code.InvalidArgument)
            }
            const {flag} = await this.flags.openFlag(request)
            const drawn = new Flag({
                ...flag,
                sold: new Sold({...flag?.sold, picture: new Picture({...flag?.sold?.picture, url: URL.createObjectURL(file)})}),
            })
            this.flags.listed = this.flags.listed.map(each => each.code === drawn.code ? drawn : each)
            return new OpenFlagResponse({flag: drawn})
        },
        addTime: this.flags.addTime.bind(this.flags),
        closeFlag: this.flags.closeFlag.bind(this.flags),
        listSubmissions: this.flags.listSubmissions.bind(this.flags),
        approveSubmission: this.flags.approveSubmission.bind(this.flags),
        rejectSubmission: this.flags.rejectSubmission.bind(this.flags),
        openSubmittedFlag: this.flags.openSubmittedFlag.bind(this.flags),
        renameFlag: this.flags.renameFlag.bind(this.flags),
    }

    readonly demoProgression = {
        getProgress: (request: PartialMessage<AdminGetProgressRequest>) =>
            this.answer("getProgress", request, this.progressOf(request.accountId ?? "")),
        reconcileTitles: this.progression.reconcileTitles.bind(this.progression),
        backfillDaysPlayed: this.progression.backfillDaysPlayed.bind(this.progression),
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
        warnPlayer: (request: PartialMessage<WarnPlayerRequest>) => {
            const inspection = this.inspectionOf(request)
            const already = inspection.warnedAt !== undefined
            if (!already) inspection.warnedAt = at(Date.now())
            return this.answer("warnPlayer", request, new WarnPlayerResponse({
                accountId: request.accountId, warnedAt: inspection.warnedAt, alreadyWarned: already,
                acknowledgedAt: inspection.warningAcknowledgedAt,
            }))
        },
        getRules: this.antibot.getRules.bind(this.antibot),
        saveRuleSet: this.antibot.saveRuleSet.bind(this.antibot),
    }
}

function muteKeyOf(request: {accountId?: string, kind?: MuteKind}): string {
    return `${request.kind ?? MuteKind.CHAT}:${request.accountId ?? ""}`
}

// The tiles of four drawings of apps/backend/internal/vandalsquad/internal/swastikas/testdata/drawings.json, on the
// map the page bundles: nov6-447, final-battle-2026-10-31T22:59-br, season0-2233 (judged none) and nov6-283.
const DEMO_GRAFFITI: {tiles: number[], flag: string, hoursAgo: number, painter: number, place: [number, number]}[] = [
    {
        tiles: [
            25533, 25534, 25535, 25536, 25539, 25687, 25690, 25836, 25839, 25987, 25990, 26134, 26135, 26136, 26137, 26138,
            26139, 26283, 26286, 26434, 26437, 26584, 26586, 26724, 26726, 26727, 26728, 26729, 26862,
        ],
        flag: "de", hoursAgo: 0.3, painter: 4, place: [46.97, 55.23],
    },
    {
        tiles: [
            251102, 251105, 251106, 251107, 251108, 251170, 251173, 251238, 251241, 251305, 251306, 251307, 251308, 251309,
            251310, 251311, 251375, 251378, 251442, 251445, 251505, 251506, 251507, 251508, 251511,
        ],
        flag: "de", hoursAgo: 3, painter: 4, place: [-28.13, -52.64],
    },
    {
        tiles: [
            210609, 210610, 210707, 210708, 210808, 210809, 210810, 210904, 210907, 210910, 211004, 211005, 211006, 211105,
            211106, 211206, 211207,
        ],
        flag: "fr", hoursAgo: 26, painter: 9, place: [-81.79, 84.83],
    },
    {
        tiles: [
            171359, 171491, 171621, 171623, 171752, 171753, 171755, 171883, 171885, 171886, 172013, 172016, 172145, 172146,
            172148, 172275, 172277, 172278, 172404, 172406, 172532, 172661,
        ],
        flag: "es", hoursAgo: 50, painter: 15, place: [-5.27, -66.33],
    },
]

const IMAGE_TYPES = ["image/jpeg", "image/png", "image/webp"]
const FLAG_PICTURE_BYTES = 5 * 1024 * 1024
const BAVARIA = "r-demobavar1"

// Made-up sold flags, each picture drawn in the page, each pinned on its capital: Bavaria and Cornwall open, Savoy
// closed 3 days ago (time added opens it again), Tyrol closed 41 days ago (it no longer comes back).
function flagSvgOf(width: number, height: number, shapes: string): string {
    return `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${width} ${height}">${shapes}</svg>`)}`
}

const MUNICH = {latitude: 48.1374, longitude: 11.5755}
const TRURO = {latitude: 50.2632, longitude: -5.051}
const INNSBRUCK = {latitude: 47.2692, longitude: 11.4041}
const RENNES = {latitude: 48.1173, longitude: -1.6778}

const LOZENGES = `<defs><pattern id="l" width="1" height="1" patternUnits="userSpaceOnUse" patternTransform="rotate(-30)">` +
    `<rect width="1" height="1" fill="#fff"/><rect width=".5" height=".5" fill="#0098d4"/><rect x=".5" y=".5" width=".5" height=".5" fill="#0098d4"/>` +
    `</pattern></defs><rect width="5" height="3" fill="url(#l)"/>`

const DEMO_FLAGS = [
    soldFlag(BAVARIA, "Bavaria", new Date(NOW + 12 * DAY), undefined, flagSvgOf(5, 3, LOZENGES), MUNICH),
    soldFlag("xb", "Brittany", new Date(NOW + 27 * DAY), undefined, flagSvgOf(27, 18,
        `<rect width="27" height="18" fill="#fff"/>` +
        [0, 4, 8, 12, 16].map(y => `<rect x="${y < 10 ? 12 : 0}" y="${y}" width="27" height="2" fill="#000"/>`).join("") +
        [[2, 2], [5, 2], [8, 2], [3.5, 5], [6.5, 5], [2, 8], [5, 8], [8, 8]]
            .map(([x, y]) => `<path d="M${x} ${y}l-.8 1.6h1.6z" fill="#000"/>`).join("")), RENNES),
    soldFlag("r-democornw1", "Cornwall", new Date(NOW + 20 * HOUR), undefined, flagSvgOf(5, 3,
        `<rect width="5" height="3" fill="#000"/><rect x="2" width="1" height="3" fill="#fff"/><rect y="1" width="5" height="1" fill="#fff"/>`), TRURO),
    soldFlag("r-demosavoy1", "Savoy", new Date(NOW + 9 * DAY), new Date(NOW - 3 * DAY), flagSvgOf(1, 1,
        `<rect width="1" height="1" fill="#d52b1e"/><rect x=".4" y=".15" width=".2" height=".7" fill="#fff"/><rect x=".15" y=".4" width=".7" height=".2" fill="#fff"/>`),
    {latitude: 45.5646, longitude: 5.9178}),
    soldFlag("r-demotyrol1", "Tyrol", new Date(NOW - 41 * DAY), new Date(NOW - 41 * DAY), flagSvgOf(2, 1,
        `<rect width="2" height="1" fill="#fff"/><rect y=".5" width="2" height=".5" fill="#c8102e"/>`), INNSBRUCK),
]

// A photo, as a player might send one for a flag: a lake at dusk, its sky, its hills and the light on the water.
function photoOf(): string {
    const roll = random(7)
    const ripples = Array.from({length: 260}, () => {
        const x = Math.round(roll() * 768), y = 300 + Math.round(roll() * 212)
        const light = Math.round(120 + roll() * 135)
        return `<ellipse cx="${x}" cy="${y}" rx="${4 + Math.round(roll() * 18)}" ry="${1 + Math.round(roll() * 3)}" ` +
            `fill="rgb(${light},${Math.round(light * 0.75)},${Math.round(90 + roll() * 120)})" opacity=".55"/>`
    }).join("")
    return flagSvgOf(768, 512,
        `<defs><linearGradient id="s" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#2b3a67"/>` +
        `<stop offset=".55" stop-color="#e3826b"/><stop offset="1" stop-color="#ffd59e"/></linearGradient>` +
        `<linearGradient id="w" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#e9a27c"/><stop offset="1" stop-color="#1f3350"/></linearGradient>` +
        `<radialGradient id="g"><stop offset="0" stop-color="#fff6d5"/><stop offset="1" stop-color="#ffb46b" stop-opacity="0"/></radialGradient></defs>` +
        `<rect width="768" height="512" fill="url(#s)"/><circle cx="520" cy="270" r="120" fill="url(#g)"/>` +
        `<path d="M0 250 L90 170 L170 230 L260 140 L360 240 L450 190 L560 260 L660 180 L768 240 L768 310 L0 310 Z" fill="#4b3f5c"/>` +
        `<path d="M0 280 L120 230 L230 275 L330 220 L440 280 L560 240 L680 285 L768 260 L768 310 L0 310 Z" fill="#2f2a44"/>` +
        `<rect y="300" width="768" height="212" fill="url(#w)"/>${ripples}`)
}

type DemoSubmission = {
    id: string, name: string, pin: {latitude: number, longitude: number}, hoursAgo: number, decidedHoursAgo?: number,
    picture: string, part?: PartialMessage<KeptSubmission>,
}

const DEMO_SUBMISSIONS: DemoSubmission[] = [
    {
        id: "demo-sub-catalonia", name: "Catalonia", pin: {latitude: 41.3874, longitude: 2.1686}, hoursAgo: 5,
        picture: flagSvgOf(9, 6, `<rect width="9" height="6" fill="#fcdd09"/>` +
            [1, 3, 5, 7].map(n => `<rect y="${(n * 6 / 9).toFixed(3)}" width="9" height="${(6 / 9).toFixed(3)}" fill="#da121a"/>`).join("")),
        part: {otherColorsPercent: 0},
    },
    {
        id: "demo-sub-tyrol", name: "Tyrol", pin: {latitude: 47.2654, longitude: 11.3928}, hoursAgo: 4,
        picture: flagSvgOf(5, 3, `<rect width="5" height="3" fill="#fff"/><rect y="1.5" width="5" height="1.5" fill="#c8102e"/>` +
            `<path d="M2.1 .6 h.8 v.9 q0 .6 -.4 .8 q-.4 -.2 -.4 -.8z" fill="#c8102e" stroke="#fff" stroke-width=".08"/>`),
        part: {otherColorsPercent: 3},
    },
    {
        id: "demo-sub-garda", name: "Lake Garda", pin: {latitude: 45.6406, longitude: 10.6763}, hoursAgo: 3, picture: photoOf(),
        part: {otherColorsPercent: 41},
    },
    {
        id: "demo-sub-bavaria", name: "Free State of Bavaria", pin: MUNICH, hoursAgo: 1,
        picture: flagSvgOf(5, 3, `${LOZENGES}<path d="M2.1 .7 h.8 v.8 q0 .6 -.4 .8 q-.4 -.2 -.4 -.8z" fill="#fff" stroke="#0098d4" stroke-width=".06"/>` +
            `<path d="M2.25 .55 l.1 -.2 l.15 .12 l.15 -.12 l.1 .2z" fill="#f5c400"/>`),
        part: {change: true, flag: BAVARIA, otherColorsPercent: 5},
    },
    {
        id: "demo-sub-sardinia", name: "Sardinia", pin: {latitude: 39.2238, longitude: 9.1217}, hoursAgo: 52, decidedHoursAgo: 30,
        picture: flagSvgOf(30, 20, `<rect width="30" height="20" fill="#fff"/><rect x="13.5" width="3" height="20" fill="#d1001f"/>` +
            `<rect y="8.5" width="30" height="3" fill="#d1001f"/>` +
            [[6.75, 4.25], [23.25, 4.25], [6.75, 15.75], [23.25, 15.75]].map(([x, y]) =>
                `<circle cx="${x}" cy="${y}" r="2.6" fill="#111"/><rect x="${x - 2.6}" y="${y - 1.4}" width="5.2" height=".8" fill="#fff"/>`).join("")),
        part: {state: SubmissionState.APPROVED, otherColorsPercent: 2},
    },
    {
        id: "demo-sub-brittany", name: "Brittany", pin: {latitude: 48.1173, longitude: -1.6778}, hoursAgo: 75, decidedHoursAgo: 70,
        picture: flagSvgOf(27, 18, `<rect width="27" height="18" fill="#fff"/>` +
            [0, 4, 8, 12, 16].map(y => `<rect y="${y}" width="27" height="2" fill="#000"/>`).join("") +
            `<rect width="11" height="10" fill="#fff"/>` +
            [[2, 2], [5.5, 2], [9, 2], [3.75, 5], [7.25, 5], [2, 8], [5.5, 8], [9, 8]].map(([x, y]) =>
                `<path d="M${x} ${y - 1} l.6 1.4 l-.6 .6 l-.6 -.6z" fill="#000"/>`).join("")),
        part: {state: SubmissionState.REJECTED, rejection: new Rejection({reason: RejectionReason.EXISTS}), otherColorsPercent: 1},
    },
    {
        id: "demo-sub-cornwall", name: "Cornwall", pin: TRURO, hoursAgo: 24 * 32, decidedHoursAgo: 24 * 30,
        picture: flagSvgOf(5, 3, `<rect width="5" height="3" fill="#000"/><rect x="2" width="1" height="3" fill="#fff"/><rect y="1" width="5" height="1" fill="#fff"/>`),
        part: {state: SubmissionState.OPENED, flag: "r-democornw1", otherColorsPercent: 0},
    },
]

// Made-up backgrounds: a gradient and a few shapes each, drawn in the page.
function svgOf(top: string, bottom: string, shapes: string): string {
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 400"><defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="1">` +
        `<stop offset="0" stop-color="${top}"/><stop offset="1" stop-color="${bottom}"/></linearGradient></defs>` +
        `<rect width="300" height="400" fill="url(#g)"/>${shapes}</svg>`
    return `data:image/svg+xml,${encodeURIComponent(svg)}`
}

const DEMO_BACKGROUNDS = [
    ["demo-war", "The war", svgOf("#1d2b53", "#7e2553", `<circle cx="150" cy="260" r="90" fill="#ff004d" opacity=".7"/><circle cx="90" cy="300" r="40" fill="#ffa300"/>`)],
    ["demo-points", "The points", svgOf("#0b3d2e", "#1d2b53", `<rect x="60" y="220" width="50" height="120" fill="#ffec27"/><rect x="125" y="180" width="50" height="160" fill="#ffec27"/><rect x="190" y="250" width="50" height="90" fill="#ffec27"/>`)],
    ["demo-rally", "The rally", svgOf("#5f574f", "#000000", `<path d="M150 170 L180 260 L270 260 L195 310 L225 390 L150 340 L75 390 L105 310 L30 260 L120 260 Z" fill="#ff77a8"/>`)],
].map(([backgroundId, name, url]) => new PromoBackground({
    backgroundId, name, contentType: "image/svg+xml", sizeBytes: BigInt(url.length), keptAt: at(Date.now() - 3 * 24 * HOUR), url,
}))
