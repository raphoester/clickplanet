import {Code, ConnectError} from "@connectrpc/connect"
import {AntibotAdmin, PlayerAdmin, VandalsquadAdmin} from "../clients.ts"
import {Graffiti} from "../gen/grpc/vandalsquad/v1/admin_pb.ts"
import {Reading, readingOf} from "../players/actions.ts"

// What the page read: the graffiti, newest first, and each painter's name, or that the squad is off on this server.
export type Listed =
    | {on: true, graffiti: Graffiti[], names: Map<string, string> | undefined}
    | {on: false}

// The squad off (its service absent) reads as such, not as a failure. Names that cannot be read leave the list listed,
// with `names` unset.
export async function listOf(vandalsquad: VandalsquadAdmin, player: PlayerAdmin, signal: AbortSignal): Promise<Listed> {
    let graffiti: Graffiti[]
    try {
        graffiti = (await vandalsquad.listGraffiti({}, {signal})).graffiti
    } catch (err) {
        if (ConnectError.from(err).code === Code.Unimplemented) return {on: false}
        throw err
    }
    const accounts = paintersOf(graffiti)
    if (accounts.length === 0) return {on: true, graffiti, names: new Map()}
    try {
        const answer = await player.getPlayers({accountIds: accounts}, {signal})
        return {on: true, graffiti, names: new Map(answer.players.map(each => [each.accountId, each.name]))}
    } catch (err) {
        if (signal.aborted) throw err
        return {on: true, graffiti, names: undefined}
    }
}

// Each painter once, in the list's order. A forgotten player's graffiti has none.
export function paintersOf(graffiti: Graffiti[]): string[] {
    return [...new Set(graffiti.map(each => each.accountId).filter(id => id !== ""))]
}

// Where each painter stands with the antibot, by account; one that could not be read is left out.
export async function standingsOf(antibot: AntibotAdmin, accounts: string[], signal: AbortSignal): Promise<Map<string, Reading>> {
    const read = await Promise.allSettled(accounts.map(accountId => readingOf(antibot, {accountId}, signal)))
    const standings = new Map<string, Reading>()
    read.forEach((each, i) => {
        if (each.status === "fulfilled") standings.set(accounts[i], each.value)
    })
    return standings
}

// A graffiti has no id: the squad tells one from another by its painter, its flag and its tiles, and keeps its first
// flagging when it sees it again, so those three name it in the address bar.
export function keyOf(graffiti: Graffiti): string {
    return `${graffiti.flaggedAt?.toDate().getTime() ?? 0}-${graffiti.countryId}-${graffiti.accountId}`
}

// Whether the painter is banned now, as the list shows it.
export function bannedOf(reading: Reading | undefined): boolean | undefined {
    if (reading === undefined || !reading.on) return undefined
    return reading.value.banned
}

export function tilesOf(graffiti: Graffiti): string {
    return graffiti.tileIds.length === 1 ? "1 tile" : `${graffiti.tileIds.length} tiles`
}

// A painter as the list names it: an account with no name is most likely a script, and a forgotten player's graffiti
// has no account left.
export function nameOf(accountId: string, names: Map<string, string> | undefined): string {
    if (accountId === "") return "A deleted account"
    return names?.get(accountId) || "Unnamed"
}
