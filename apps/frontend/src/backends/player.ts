/**
 * A player's username. Every call answers for the account the click token
 * names, so each one needs a session, and only a signed-in player asks.
 */

import {GUEST_PREFIX} from "./chat.ts"

export const MIN_USERNAME_LENGTH = 3
export const MAX_USERNAME_LENGTH = 15

const USERNAME_CHARACTERS = /^[\p{L}\p{Mn}\p{Mc}\p{Nd}_ ]+$/u
const INVISIBLE = /[\p{Default_Ignorable_Code_Point}\p{Variation_Selector}]/u
/** A mark after anything but a letter or a mark, or more than three in a row. */
const STRAY_MARKS = /(^|[^\p{L}\p{M}])\p{M}|\p{M}{4}/u
/** The scripts whose letters look alike. The server refuses every other mix too. */
const LOOKALIKE_SCRIPTS = [/\p{Script=Latin}/u, /\p{Script=Greek}/u, /\p{Script=Cyrillic}/u]

/**
 * The name as the server keeps it: in NFC, the spaces at its ends cut.
 */
export function usernameOf(typed: string): string {
    return typed.normalize("NFC").replace(/^ +| +$/g, "")
}

/**
 * Mirrors `player.v1.PlayerService/SetName`'s rule, for the input, on a name
 * from `usernameOf`: 3 to 15 code points, letters of any script, marks after a
 * letter, digits, underscores and single spaces, no Latin, Greek and Cyrillic
 * letters mixed, and not starting with the prefix the chat puts before every
 * guest's name. The server is the authority: it also refuses other mixes of
 * scripts, and it alone knows whether another account holds the name.
 */
export function isValidUsername(name: string): boolean {
    const length = [...name].length
    return length >= MIN_USERNAME_LENGTH
        && length <= MAX_USERNAME_LENGTH
        && USERNAME_CHARACTERS.test(name)
        && !INVISIBLE.test(name)
        && !STRAY_MARKS.test(name)
        && !name.includes("  ")
        && name === usernameOf(name)
        && LOOKALIKE_SCRIPTS.filter((script) => script.test(name)).length <= 1
        && !folded(name).startsWith(GUEST_PREFIX)
}

/** Close to the server's case fold: enough to see the guest prefix under any case or width. */
function folded(name: string): string {
    return name.normalize("NFKC").toLowerCase()
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

/**
 * What a player says about itself when it announces that it is playing. The
 * name is not in it: the server reads it off the account the token names.
 */
export type Presence = {
    countryCode: string
}

/** One player as the roster and the chat show it: enough to open its card. */
export type PlayerLine = {
    /** A username, or `GUEST_PREFIX` and the guest's code: ready to show, and the chat's name for it. */
    name: string
    countryCode: string
    guest: boolean
    /** An admin of the game. Never a guest. */
    admin: boolean
}

/** One player on the roster, as the server names it. */
export type RosterEntry = PlayerLine & {
    /** Names the line while the player stays on: through a new flag, a sign-in and a new name. */
    key: string
}

/** One event of the live roster. */
export type RosterEvent =
    /** The whole roster, in the server's order. The first event of every connection. */
    | {kind: "roster", entries: RosterEntry[]}
    /** A player joined, or its line changed: it replaces the line with the same key. */
    | {kind: "entry", entry: RosterEntry}
    | {kind: "left", key: string}

/**
 * Who is playing: `player.v1.PlayerService/Announce`, `Leave` and `ListenForEvents`. Kept
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
     * Says the page is closing, with the token held, if any. Sent with
     * `keepalive` so it outlives the page, and nothing waits on it.
     */
    leave(): void

    /**
     * Follows the live roster until the answer is called, reconnecting on its
     * own; every connection starts with a `roster` event. `onUnavailable` is
     * called once, and nothing after, when the server has no live roster.
     */
    listenForRoster(onEvent: (event: RosterEvent) => void, onUnavailable: () => void): () => void
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
    admin: boolean
}

/** `player.v1.PlayerService/GetPlayer`. Needs no session. */
export interface PlayerInfoBackend {
    /** Undefined when no player holds the name, in any case. */
    playerInfo(name: string): Promise<PlayerInfo | undefined>
}
