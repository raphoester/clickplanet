import {Country} from "../domain/countries.ts"

// What the page asks CrazyGames, through its SDK. Off CrazyGames there is none.
export interface CrazyGamesPage {
    user: CrazyGamesUser
    room: CrazyGamesRoom

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

// The whole world is one room, always open: friends who join are asked to play for the flag.
export interface CrazyGamesRoom {
    report(flag: string): void

    // The invite the page was opened from, then each invite taken while it plays.
    listenForInvites(listener: (flag: Country) => void): () => void

    inviteLink: InviteLink
}

// undefined when CrazyGames gives no link.
export type InviteLink = (flag: string) => Promise<string | undefined>

export type CrazyGamesSettings = {
    chat: boolean
    audio: boolean
}

export const DEFAULT_CRAZYGAMES_SETTINGS: CrazyGamesSettings = {chat: true, audio: true}
