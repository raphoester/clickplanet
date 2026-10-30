import {Landing} from "./landing.ts"

// CrazyGames frames the game from this host, so clickplanet.lol never runs as the CrazyGames page.
export const CRAZYGAMES_HOST = "crazygames.clickplanet.lol"

export const CRAZYGAMES_TAG = "cg"

export function onCrazyGames(hostname: string, buildFlag: string | undefined): boolean {
    return buildFlag === "1" || hostname === CRAZYGAMES_HOST
}

// Every account made there came from CrazyGames, whatever its link said.
export function crazyGamesLanding(seen: Landing): Landing {
    return {...seen, tag: CRAZYGAMES_TAG}
}
