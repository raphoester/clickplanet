// What the page asks CrazyGames, through its SDK. Off CrazyGames there is none.
export interface CrazyGamesPage {
    user: CrazyGamesUser
    ads: RewardedAds

    settings(): CrazyGamesSettings

    listenForSettings(listener: (settings: CrazyGamesSettings) => void): () => void

    loaded(): void

    playing(on: boolean): void
}

export interface CrazyGamesUser {
    // The signed token of the CrazyGames account the player is signed in to; undefined for a guest.
    token(): Promise<string | undefined>

    // CrazyGames' own sign-in: true once the player is signed in, false when they closed it.
    prompt(): Promise<boolean>

    listenForSignIn(listener: () => void): () => void
}

// watched: the ad played to its end. off: no ad in this session at all (the Basic Launch, the kids' site).
export type AdOutcome = "watched" | "unavailable" | "blocked" | "off"

export interface RewardedAds {
    blocked(): Promise<boolean>

    // onStart runs when the ad starts playing, not when it is asked for: the game is muted then.
    watch(onStart: () => void): Promise<AdOutcome>
}

export type CrazyGamesSettings = {
    chat: boolean
    audio: boolean
}

export const DEFAULT_CRAZYGAMES_SETTINGS: CrazyGamesSettings = {chat: true, audio: true}
