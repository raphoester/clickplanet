/**
 * A player's username. Every call answers for the account the click token
 * names, so each one needs a session, and only a signed-in player asks.
 */

import {GUEST_PREFIX} from "./chat.ts"

export const MIN_USERNAME_LENGTH = 3
export const MAX_USERNAME_LENGTH = 20

const USERNAME_CHARACTERS = /^[A-Za-z0-9_]+$/

/**
 * Mirrors `player.v1.PlayerService/SetName`'s rule, for the input: 3 to 20
 * ASCII letters, digits or underscores, and not starting with the prefix the
 * chat puts before every guest's name. The server is the authority, and it
 * alone knows whether another account holds the name.
 */
export function isValidUsername(name: string): boolean {
    return name.length >= MIN_USERNAME_LENGTH
        && name.length <= MAX_USERNAME_LENGTH
        && USERNAME_CHARACTERS.test(name)
        && !name.toLowerCase().startsWith(GUEST_PREFIX)
}

export type Profile = {
    accountId: string
    /** The username, as it was typed. Empty while the player has not chosen one. */
    name: string
}

export interface PlayerBackend {
    profile(): Promise<Profile>

    /** Answers the profile as the server stored it. */
    setName(name: string): Promise<Profile>
}

/** Why the server said no. One class, like `AuthError`: each is one line under the form. */
export type PlayerFailure =
    /** The name breaks the rule. */
    | "invalid"
    /** Another account holds the name, ignoring case. */
    | "taken"
    /** No session could name an account, even after a fresh one. */
    | "notSignedIn"
    /** The account has no provider linked: only a signed-in player picks a username. */
    | "guest"
    /** Anything else: the network, the server, a session that could not be minted. */
    | "failed"

export class PlayerError extends Error {
    constructor(public readonly failure: PlayerFailure, options?: {cause?: unknown}) {
        super(`player request refused: ${failure}`, options)
        this.name = "PlayerError"
    }
}

export function playerFailureOf(e: unknown): PlayerFailure {
    return e instanceof PlayerError ? e.failure : "failed"
}

/** What a player says about itself when it announces that it is playing. */
export type Presence = {
    countryCode: string
    /**
     * The name the guest typed in the chat, without `GUEST_PREFIX`. Sent by a
     * player with a username too: the server ignores it then.
     */
    guestName: string
}

/** One player on the roster, as the server names it. */
export type RosterEntry = {
    /** A username, or `GUEST_PREFIX` and the typed name or the tag: ready to show. */
    name: string
    /** As on a chat message: the salted hash of the address it announced from. */
    tag: string
    countryCode: string
    guest: boolean
}

/**
 * Who is playing: `player.v1.PlayerService/Announce` and `GetRoster`. Kept
 * apart from `PlayerBackend`, which is the account panel's: the two are used by
 * different parts of the page, and a fake of one need not fake the other.
 */
export interface PresenceBackend {
    /**
     * The click token an announce would carry, already in hand, or undefined.
     * Never mints. The presence schedule watches it to announce as soon as a
     * click has minted one, and again when it changes to another account's.
     */
    heldSession(): string | undefined

    /**
     * Resolves false, sending nothing, when no token is held. A refusal throws
     * a `PlayerError`; one refused for its session drops the token and is not
     * sent again with a fresh one — that would mint.
     */
    announce(presence: Presence): Promise<boolean>

    /**
     * Everyone playing, in the server's order: players with a username, then
     * guests, each by name ignoring case. Throws `RosterUnavailableError` when
     * the server has no roster at all.
     */
    roster(): Promise<RosterEntry[]>
}

/** The server does not serve a roster: one from before it, or with no player module. */
export class RosterUnavailableError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("the server has no roster", options)
        this.name = "RosterUnavailableError"
    }
}

/** What anybody may know about a player with a username. A guest has none of it. */
export type PlayerInfo = {
    /** The username, as its player typed it. */
    name: string
    tilesTaken: number
    /** Days in a row, UTC, with a tile taken. 0 once a whole day went by without one. */
    streakCurrent: number
    streakBest: number
    /** When the account was made, in ms. Undefined when the server does not know. */
    createdAt?: number
}

/** `player.v1.PlayerService/GetPlayer`. Needs no session. */
export interface PlayerInfoBackend {
    /** Undefined when no player holds the name, in any case. */
    playerInfo(name: string): Promise<PlayerInfo | undefined>
}
