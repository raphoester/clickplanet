// What the page asks CrazyGames, through its SDK. Off CrazyGames there is none.
export interface CrazyGamesPage {
    user: CrazyGamesUser

    settings(): CrazyGamesSettings

    listenForSettings(listener: (settings: CrazyGamesSettings) => void): () => void

    // The game is playable: the load is over and the gameplay starts. It never pauses.
    started(): void
}

export interface CrazyGamesUser {
    // The signed token of the CrazyGames account the player is signed in to; undefined for a guest.
    token(): Promise<string | undefined>

    // CrazyGames' own sign-in: true once the player is signed in, false when they closed it.
    prompt(): Promise<boolean>

    listenForSignIn(listener: () => void): () => void
}

export type CrazyGamesSettings = {
    chat: boolean
    audio: boolean
}

export const DEFAULT_CRAZYGAMES_SETTINGS: CrazyGamesSettings = {chat: true, audio: true}
