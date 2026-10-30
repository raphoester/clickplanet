import {Landing} from "./landing.ts"
import {Countries, Country} from "./countries.ts"
import {SHARE_FLAG_PARAM} from "./shareCard.ts"

// CrazyGames frames the game from this host, so clickplanet.lol never runs as the CrazyGames page.
export const CRAZYGAMES_HOST = "crazygames.clickplanet.lol"

export const CRAZYGAMES_TAG = "cg"

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
