import {Duration} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {LONG_CALL_MS} from "../api.ts"
import {AlertsAdmin, AntibotAdmin, ChatAdmin, PlanetAdmin, PlayerAdmin, ProgressionAdmin} from "../clients.ts"
import {DiscordLink} from "../gen/grpc/alerts/v1/admin_pb.ts"
import {MuteKind} from "../gen/grpc/chat/v1/chat_pb.ts"
import {InspectPlayerResponse} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {Player} from "../gen/grpc/planet/v1/admin_pb.ts"
import {ChargesHeld} from "../gen/grpc/planet/v1/planet_pb.ts"
import {AdminGetPlayerResponse, PlayerMatch} from "../gen/grpc/player/v1/admin_pb.ts"
import {Player as Identity} from "../gen/grpc/player/v1/player_pb.ts"
import {AdminGetProgressResponse} from "../gen/grpc/progression/v1/admin_pb.ts"
import {Stats} from "../gen/grpc/progression/v1/progression_pb.ts"
import {timeOf} from "../components/time.ts"
import {Meanings, wordsOf} from "../refusal.ts"
import {Caller} from "./caller.ts"

// The server's own default for each: an hour of mute, and the ban ladder's next step.
export const DEFAULT_MUTE_SECONDS = 60 * 60
export const LADDER = 0

// The server lists 20 when asked for none.
export const LIMITS = [20, 50, 100, 500]

export type Muted = {until: Date}
export type Unmuted = {until: Date}
// The mute that holds an account now, or none. `byNetwork`: it was given to another account on its network.
export type Muting = {until: Date, byNetwork: boolean} | undefined
// Each kind holds apart: an account may be muted from the whole chat, from its GIFs, or both.
export type Mutings = {chat: Muting, gifs: Muting}
export const UNMUTED: Mutings = {chat: undefined, gifs: undefined}
export type Banned = {until: Date, offence: number, enforced: boolean}
export type Unbanned = {offence: number, until: Date | undefined, pardonedUntil: Date | undefined}
export type Reverted = {touched: number, held: number, restored: number}
export type Grant = {refill: boolean, bomb: boolean, enclosures: number, spreadClicks: number, shields: number}
export type Granted = {before: ChargesHeld, after: ChargesHeld}
export type Released = {locked: boolean}
export type Players = {players: Player[], total: number}
export type Identities = Map<string, Identity>
// Who each account is. `unnamed` says why there are no names, when they could not be read.
export type Named = {identities: Identities, unnamed: string | undefined}
// A page of players and who each account is.
export type Listed = Players & Named
export type Matches = {players: PlayerMatch[], total: number}
// No identity: an account with no name and no guest code, most likely a script that clicks without announcing.
export type Profile = {identity: AdminGetPlayerResponse | undefined, progress: AdminGetProgressResponse}

export const MOST = 99
export const NO_GRANT: Grant = {refill: false, bomb: false, enclosures: 0, spreadClicks: 0, shields: 0}
// The server caps each count at its stack, so this fills every charge and the answer says what fit.
export const MAX_GRANT: Grant = {refill: true, bomb: true, enclosures: MOST, spreadClicks: MOST, shields: MOST}

// What each kind of mute takes away, as the page says it: its choice, the account it holds in a sentence, and what
// was done. A mute of the GIFs leaves the player its text and its reactions.
export const MUTE_KINDS: {kind: MuteKind, label: string, of: (who: string) => string, muted: string, unmuted: string}[] = [
    {kind: MuteKind.CHAT, label: "The whole chat", of: who => who, muted: "Muted", unmuted: "Unmuted"},
    {kind: MuteKind.GIFS, label: "GIFs alone", of: who => `the GIFs of ${who}`, muted: "GIFs muted", unmuted: "GIFs unmuted"},
]

export function muteWordsOf(kind: MuteKind) {
    return MUTE_KINDS.find(each => each.kind === kind) ?? MUTE_KINDS[0]
}

// A mute is an account's: the chat has no scope to mute.
export async function mute(chat: ChatAdmin, accountId: string, seconds: number, kind = MuteKind.CHAT): Promise<Muted> {
    const answer = await chat.mute({accountId, duration: new Duration({seconds: BigInt(seconds)}), kind})
    return {until: dateOf(answer.mutedUntil)}
}

// Ends every mute of the kind given to the account. A mute given to another account on its network still holds it.
export async function unmute(chat: ChatAdmin, accountId: string, kind = MuteKind.CHAT): Promise<Unmuted> {
    const answer = await chat.unmute({accountId, kind})
    return {until: dateOf(answer.mutedUntil)}
}

export async function mutingOf(
    chat: ChatAdmin, accountId: string, kind = MuteKind.CHAT, signal?: AbortSignal,
): Promise<Muting> {
    const answer = await chat.getMute({accountId, kind}, {signal})
    if (answer.mutedUntil === undefined) return undefined
    return {until: answer.mutedUntil.toDate(), byNetwork: answer.mutedAccountId !== accountId}
}

export async function mutingsOf(chat: ChatAdmin, accountId: string, signal?: AbortSignal): Promise<Mutings> {
    const [whole, gifs] = await Promise.all([
        mutingOf(chat, accountId, MuteKind.CHAT, signal),
        mutingOf(chat, accountId, MuteKind.GIFS, signal),
    ])
    return {chat: whole, gifs}
}

export async function ban(antibot: AntibotAdmin, caller: Caller, seconds: number): Promise<Banned> {
    const duration = seconds === LADDER ? undefined : new Duration({seconds: BigInt(seconds)})
    const answer = await antibot.banPlayer({...caller, duration})
    return {until: dateOf(answer.bannedUntil), offence: answer.offence, enforced: answer.enforced}
}

// Lifts the latest ban, running or ended, and forgets its offence. One call forgets one offence.
export async function unban(antibot: AntibotAdmin, caller: Caller): Promise<Unbanned> {
    const answer = await antibot.unbanPlayer(caller)
    return {offence: answer.offence, until: answer.bannedUntil?.toDate(), pardonedUntil: answer.pardonedUntil?.toDate()}
}

// Paced on the server, a tile at a time on the live stream: a big revert takes seconds.
export async function revert(planet: PlanetAdmin, caller: Caller, dryRun: boolean): Promise<Reverted> {
    const answer = await planet.revertPlayer({...caller, dryRun}, {timeoutMs: LONG_CALL_MS})
    return {touched: answer.touched, held: answer.held, restored: answer.restored}
}

export async function inspect(antibot: AntibotAdmin, caller: Caller, signal?: AbortSignal): Promise<InspectPlayerResponse> {
    return antibot.inspectPlayer(caller, {signal})
}

// The antibot's reading, or that it is off on this server (its service absent), which is no failure.
export type Reading = {on: true, value: InspectPlayerResponse} | {on: false}

export async function readingOf(antibot: AntibotAdmin, caller: Caller, signal?: AbortSignal): Promise<Reading> {
    try {
        return {on: true, value: await inspect(antibot, caller, signal)}
    } catch (err) {
        if (ConnectError.from(err).code === Code.Unimplemented) return {on: false}
        throw err
    }
}

// A charge is an account's: the caps hold, so `after` says what fit.
export async function grant(planet: PlanetAdmin, accountId: string, charges: Grant): Promise<Granted> {
    const answer = await planet.grantCharges({accountId, ...charges})
    return {before: answer.before ?? new ChargesHeld(), after: answer.after ?? new ChargesHeld()}
}

// Lifts the hour of wait after a change of flag. A lock of the rules in force still holds.
export async function release(planet: PlanetAdmin, accountId: string): Promise<Released> {
    const answer = await planet.releaseAllegiance({accountId})
    return {locked: answer.locked}
}

// Answers the name as the server kept it. The player's next own change goes at once.
export async function rename(player: PlayerAdmin, accountId: string, name: string): Promise<string> {
    const answer = await player.renamePlayer({accountId, name})
    return answer.name
}

export type Linking = {on: false} | {on: true, link: DiscordLink | undefined}

// The account's Discord link, if any. The alerts module off (its service absent) reads as such, not as a failure.
export async function linkingOf(alerts: AlertsAdmin, accountId: string, signal?: AbortSignal): Promise<Linking> {
    try {
        const answer = await alerts.getPlayerLink({accountId}, {signal})
        return {on: true, link: answer.link}
    } catch (err) {
        if (ConnectError.from(err).code === Code.Unimplemented) return {on: false}
        throw err
    }
}

// Forgets the link and takes the flag's role off the Discord user, who stays in the server. False: it was not linked.
export async function unlink(alerts: AlertsAdmin, accountId: string): Promise<boolean> {
    const answer = await alerts.unlinkPlayer({accountId})
    return answer.unlinked
}

export function grantsAnything(charges: Grant): boolean {
    return charges.refill || charges.bomb || charges.enclosures > 0 || charges.spreadClicks > 0 || charges.shields > 0
}

// Most takes first, then most tiles held, over every flag and the whole map.
export async function topPlayers(planet: PlanetAdmin, limit: number, signal?: AbortSignal): Promise<Players> {
    const answer = await planet.topPlayers({limit}, {signal})
    return {players: answer.players, total: answer.total}
}

// Who took a tile for `flag` on `area`'s ground, or anywhere with no area, latest take first.
export async function findPlayers(
    planet: PlanetAdmin, flag: string, area: string, limit: number, signal?: AbortSignal,
): Promise<Players> {
    const answer = await planet.findPlayers({flagCountryId: flag, areaCountryId: area, limit}, {signal})
    return {players: answer.players, total: answer.total}
}

// Every account whose username or guest code holds `name`: the same name first, then the names it starts.
export async function searchPlayers(player: PlayerAdmin, name: string, limit: number, signal?: AbortSignal): Promise<Matches> {
    const answer = await player.searchPlayers({name, limit}, {signal})
    return {players: answer.players, total: answer.total}
}

// Each account of a page by its username or its guest code. A page whose names cannot be read is still listed.
export async function withNames(player: PlayerAdmin, page: Players, signal?: AbortSignal): Promise<Listed> {
    return {...page, ...await namesOf(player, page.players.map(row => row.accountId), signal)}
}

// Each account by its username or its guest code, asked once. Names that cannot be read fail nothing: they say why.
export async function namesOf(player: PlayerAdmin, accounts: string[], signal?: AbortSignal): Promise<Named> {
    const accountIds = [...new Set(accounts.filter(id => id !== ""))]
    if (accountIds.length === 0) return {identities: new Map(), unnamed: undefined}

    try {
        const answer = await player.getPlayers({accountIds}, {signal})
        return {identities: new Map(answer.players.map(identity => [identity.accountId, identity])), unnamed: undefined}
    } catch (err) {
        if (signal?.aborted === true) throw err
        return {identities: new Map(), unnamed: wordsOf(err)}
    }
}

// Everything the game keeps about an account, a guest's too, or undefined when it keeps nothing.
export async function profileOf(
    player: PlayerAdmin, progression: ProgressionAdmin, accountId: string, signal?: AbortSignal,
): Promise<Profile | undefined> {
    const [identity, progress] = await Promise.all([
        identityOf(player, accountId, signal),
        progression.getProgress({accountId}, {signal}),
    ])
    if (identity === undefined && !keepsAnything(progress)) return undefined
    return {identity, progress}
}

// Progression answers zeros, never NotFound, for an account it keeps nothing about.
function keepsAnything(progress: AdminGetProgressResponse): boolean {
    const stats = progress.stats ?? new Stats()
    return stats.tilesTaken > 0n || stats.streakCurrent > 0 || stats.streakBest > 0 || stats.streakLastDay !== ""
        || progress.messagesSent > 0n || progress.titles.length > 0 || progress.wornTitle !== undefined
        || progress.playsFor.length > 0 || progress.playsAgainst.length > 0
}

async function identityOf(player: PlayerAdmin, accountId: string, signal?: AbortSignal): Promise<AdminGetPlayerResponse | undefined> {
    try {
        return await player.getPlayer({accountId}, {signal})
    } catch (err) {
        if (ConnectError.from(err).code === Code.NotFound) return undefined
        throw err
    }
}

export function bannedWords(banned: Banned): string {
    return `Banned until ${timeOf(banned.until)}, offence ${banned.offence}.`
        + (banned.enforced ? "" : " Not enforced: the antibot only watches on this server.")
}

export type Action = "mute" | "unmute" | "ban" | "unban" | "revert" | "inspect" | "grant" | "release" | "rename" | "unlink" | "list"

export const ANTIBOT_OFF = "The antibot is off on this server, so it cannot ban, unban or inspect."
export const NOT_BANNED = "This player has no ban left to lift."
export const NOT_MUTED = "This player has no mute of its own to lift."
export const ALERTS_OFF = "Discord pings are off on this server, so no account is linked to Discord."
export const NO_USERNAME = "This account has no username to change."

const MEANINGS: Record<Action, Meanings> = {
    mute: {},
    unmute: {notFound: NOT_MUTED},
    ban: {unimplemented: ANTIBOT_OFF},
    unban: {notFound: NOT_BANNED, unimplemented: ANTIBOT_OFF},
    revert: {},
    inspect: {unimplemented: ANTIBOT_OFF},
    grant: {},
    release: {},
    rename: {notFound: NO_USERNAME},
    unlink: {unimplemented: ALERTS_OFF},
    list: {},
}

// What a refusal means for the action it refused, in the operator's words.
export function refusalOf(action: Action, err: unknown): string {
    return wordsOf(err, MEANINGS[action])
}

function dateOf(at: {toDate(): Date} | undefined): Date {
    if (at === undefined) throw new Error("the server answered no time")
    return at.toDate()
}
