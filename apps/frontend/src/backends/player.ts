import {NameColor} from "../gen/grpc/player/v1/color_pb.ts"
import {GUEST_PREFIX} from "./chat.ts"
import {Progress} from "./progression.ts"

export {NameColor}

export const MIN_USERNAME_LENGTH = 3
export const MAX_USERNAME_LENGTH = 15

const USERNAME_CHARACTERS = /^[\p{L}\p{Mn}\p{Mc}\p{Nd}_ ]+$/u
const INVISIBLE = /[\p{Default_Ignorable_Code_Point}\p{Variation_Selector}]/u
const STRAY_MARKS = /(^|[^\p{L}\p{M}])\p{M}|\p{M}{4}/u
const LOOKALIKE_SCRIPTS = [/\p{Script=Latin}/u, /\p{Script=Greek}/u, /\p{Script=Cyrillic}/u]

export function usernameOf(typed: string): string {
    return typed.normalize("NFC").replace(/^ +| +$/g, "")
}

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

function folded(name: string): string {
    return name.normalize("NFKC").toLowerCase()
}

export type Profile = {
    accountId: string
    name: string
    renamableAt?: number
}

export type ColoredProfile = Profile & {
    color: NameColor
}

export interface PlayerBackend {
    profile(): Promise<ColoredProfile>

    setName(name: string): Promise<Profile>

    setColor(color: NameColor): Promise<NameColor>
}

export type PlayerFailure =
    | "invalid"
    | "taken"
    | "notSignedIn"
    | "guest"
    | "unnamed"
    | "tooSoon"
    | "failed"

export class PlayerError extends Error {
    constructor(public readonly failure: PlayerFailure, options?: {cause?: unknown}) {
        super(`player request refused: ${failure}`, options)
        this.name = "PlayerError"
    }
}

export class RenameTooSoonError extends PlayerError {
    constructor(public readonly renamableAt: number, options?: {cause?: unknown}) {
        super("tooSoon", options)
        this.name = "RenameTooSoonError"
    }
}

export function playerFailureOf(e: unknown): PlayerFailure {
    return e instanceof PlayerError ? e.failure : "failed"
}

export type Presence = {
    countryCode: string
}

export type PlayerLine = {
    // Empty for the author of a chat message kept before accounts, or whose account is deleted.
    accountId: string
    name: string
    countryCode: string
    guest: boolean
    admin: boolean
    color: NameColor
}

export type RosterEntry = PlayerLine & {
    key: string
}

export type RosterEvent =
    | {kind: "roster", entries: RosterEntry[]}
    | {kind: "entry", entry: RosterEntry}
    | {kind: "left", key: string}

export interface PresenceBackend {
    heldSession(): string | undefined

    announce(presence: Presence): Promise<boolean>

    leave(): void

    listenForRoster(onEvent: (event: RosterEvent) => void, onUnavailable: () => void): () => void
}

// A guest is its name alone.
export type Player = {
    name: string
    guest: boolean
    createdAt?: number
    admin: boolean
    color: NameColor
}

export interface PlayerReader {
    player(accountId: string): Promise<Player | undefined>
}

export type PlayerInfo = Player & Progress

export interface PlayerInfoBackend {
    playerInfo(accountId: string): Promise<PlayerInfo | undefined>
}
