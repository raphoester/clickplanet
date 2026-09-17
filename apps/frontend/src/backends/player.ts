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
