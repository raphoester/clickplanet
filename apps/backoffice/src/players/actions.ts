import {Duration} from "@bufbuild/protobuf"
import {Code, ConnectError} from "@connectrpc/connect"
import {LONG_CALL_MS} from "../api.ts"
import {AntibotAdmin, ChatAdmin, PlanetAdmin, PlayerAdmin} from "../clients.ts"
import {timeOf} from "../components/time.ts"
import {InspectPlayerResponse} from "../gen/grpc/antibot/v1/admin_pb.ts"
import {Player} from "../gen/grpc/planet/v1/admin_pb.ts"
import {ChargesHeld} from "../gen/grpc/planet/v1/planet_pb.ts"
import {AdminGetPlayerResponse, PlayerMatch} from "../gen/grpc/player/v1/admin_pb.ts"
import {Author} from "../gen/grpc/player/v1/internal_pb.ts"
import {Meanings, wordsOf} from "../refusal.ts"
import {Caller} from "./caller.ts"

// The server's own default for each: an hour of mute, and the ban ladder's next step.
export const DEFAULT_MUTE_SECONDS = 60 * 60
export const LADDER = 0

// The server lists 20 when asked for none.
export const LIMITS = [20, 50, 100, 500]

export type Muted = {until: Date}
export type Banned = {until: Date, offence: number, enforced: boolean}
export type Unbanned = {offence: number, until: Date | undefined, pardonedUntil: Date | undefined}
export type Reverted = {touched: number, held: number, restored: number}
export type Grant = {refill: boolean, bomb: boolean, enclosures: number, spreadClicks: number, shields: number}
export type Granted = {before: ChargesHeld, after: ChargesHeld}
export type Players = {players: Player[], total: number}
export type Authors = Map<string, Author>
// Who each account is. `unnamed` says why there are no names, when they could not be read.
export type Named = {authors: Authors, unnamed: string | undefined}
// A page of players and who each account is.
export type Listed = Players & Named
export type Matches = {players: PlayerMatch[], total: number}
export type Profile = AdminGetPlayerResponse

export const NO_GRANT: Grant = {refill: false, bomb: false, enclosures: 0, spreadClicks: 0, shields: 0}

// A mute is an account's: the chat has no scope to mute.
export async function mute(chat: ChatAdmin, accountId: string, seconds: number): Promise<Muted> {
    const answer = await chat.mute({accountId, duration: new Duration({seconds: BigInt(seconds)})})
    return {until: dateOf(answer.mutedUntil)}
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

// What an unban lifted, in the operator's words.
export function unbannedWords(lifted: Unbanned): string {
    const until = lifted.until === undefined ? "" : `: its ban was until ${timeOf(lifted.until)}`
    const pardoned = lifted.pardonedUntil === undefined ? "" : ` Pardoned until ${timeOf(lifted.pardonedUntil)}.`
    return `Lifted offence ${lifted.offence}${until}.${pardoned}`
}

// Paced on the server, a tile at a time on the live stream: a big revert takes seconds.
export async function revert(planet: PlanetAdmin, caller: Caller, dryRun: boolean): Promise<Reverted> {
    const answer = await planet.revertPlayer({...caller, dryRun}, {timeoutMs: LONG_CALL_MS})
    return {touched: answer.touched, held: answer.held, restored: answer.restored}
}

export async function inspect(antibot: AntibotAdmin, caller: Caller, signal?: AbortSignal): Promise<InspectPlayerResponse> {
    return antibot.inspectPlayer(caller, {signal})
}

// A charge is an account's: the caps hold, so `after` says what fit.
export async function grant(planet: PlanetAdmin, accountId: string, charges: Grant): Promise<Granted> {
    const answer = await planet.grantCharges({accountId, ...charges})
    return {before: answer.before ?? new ChargesHeld(), after: answer.after ?? new ChargesHeld()}
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
export async function withAuthors(player: PlayerAdmin, page: Players, signal?: AbortSignal): Promise<Listed> {
    return {...page, ...await namesOf(player, page.players.map(row => row.accountId), signal)}
}

// Each account by its username or its guest code, asked once. Names that cannot be read fail nothing: they say why.
export async function namesOf(player: PlayerAdmin, accounts: string[], signal?: AbortSignal): Promise<Named> {
    const accountIds = [...new Set(accounts.filter(id => id !== ""))]
    if (accountIds.length === 0) return {authors: new Map(), unnamed: undefined}

    try {
        const answer = await player.getAuthors({accountIds}, {signal})
        return {authors: new Map(answer.authors.map(author => [author.accountId, author])), unnamed: undefined}
    } catch (err) {
        if (signal?.aborted === true) throw err
        return {authors: new Map(), unnamed: wordsOf(err)}
    }
}

// Everything the game keeps about an account, a guest's too, or undefined when it keeps nothing.
export async function profileOf(player: PlayerAdmin, accountId: string, signal?: AbortSignal): Promise<Profile | undefined> {
    try {
        return await player.getPlayer({accountId}, {signal})
    } catch (err) {
        if (ConnectError.from(err).code === Code.NotFound) return undefined
        throw err
    }
}

export type Action = "mute" | "ban" | "unban" | "revert" | "inspect" | "grant" | "list"

export const ANTIBOT_OFF = "The antibot is off on this server, so it cannot ban, unban or inspect."
export const NOT_BANNED = "This player has no ban left to lift."

const MEANINGS: Record<Action, Meanings> = {
    mute: {},
    ban: {unimplemented: ANTIBOT_OFF},
    unban: {notFound: NOT_BANNED, unimplemented: ANTIBOT_OFF},
    revert: {},
    inspect: {unimplemented: ANTIBOT_OFF},
    grant: {},
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
