import {Landing} from "./landing.ts"
import {Countries, Country} from "./countries.ts"
import {SHARE_FLAG_PARAM} from "./shareCard.ts"

// CrazyGames frames the game from this host, so clickplanet.lol never runs as the CrazyGames page.
export const CRAZYGAMES_HOST = "crazygames.clickplanet.lol"

const CRAZYGAMES_TAG = "cg"

// One world, so one room for every player.
export const WORLD_ROOM = "world"

export type InviteParams = Record<string, string>

export function onCrazyGames(hostname: string, buildFlag: string | undefined): boolean {
    return buildFlag === "1" || hostname === CRAZYGAMES_HOST
}

// Every account made there came from CrazyGames, whatever its link said.
export function crazyGamesLanding(seen: Landing): Landing {
    return {...seen, tag: CRAZYGAMES_TAG}
}

export function inviteParamsOf(flag: string): InviteParams {
    return {[SHARE_FLAG_PARAM]: flag}
}

export function invitedCountry(params: InviteParams | null): Country | undefined {
    const code = params?.[SHARE_FLAG_PARAM]
    return code ? Countries.get(code.trim().toLowerCase()) : undefined
}

// The server keeps an account its player asked to delete this long; a sign-in to it before then brings it back.
export const DELETION_GRACE_MS = 30 * 24 * 60 * 60 * 1000

// The CrazyGames user who asked, on this device, to delete their account, and when.
export type Leaver = {user: string, at: number}

export type LeaverStore = {
    read(): Leaver | undefined
    write(leaver: Leaver | undefined): void
}

// The user a CrazyGames token is for, from its payload. The server checks the token; this only tells the players of one
// device apart.
export function crazyGamesUserOf(token: string): string | undefined {
    const payload = token.split(".")[1]
    if (!payload) return undefined
    try {
        const base64 = payload.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(payload.length / 4) * 4, "=")
        const claims = JSON.parse(atob(base64)) as {userId?: unknown}
        return typeof claims.userId === "string" && claims.userId !== "" ? claims.userId : undefined
    } catch {
        return undefined
    }
}

// The page signs in as whoever CrazyGames says at every start, which would bring back at once an account its player
// just asked to delete. So the device remembers who asked: until the grace period ends, the automatic sign-in leaves
// them a guest, and only a sign-in they ask for brings the account back.
export function leaving(leaver: Leaver | undefined, user: string | undefined, now: number): boolean {
    return leaver !== undefined && user !== undefined && leaver.user === user && now - leaver.at < DELETION_GRACE_MS
}
