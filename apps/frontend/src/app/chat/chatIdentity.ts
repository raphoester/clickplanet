import {v4 as generateUUID} from 'uuid';
import {countRunes, MAX_NAME_LENGTH} from "../../backends/chat.ts";

export const CHAT_IDENTITY_STORAGE_KEY = 'clickplanet-chat-identity'

export type ChatIdentity = {
    authorId: string
    name: string
}

export function parseStoredIdentity(raw: string | null | undefined): ChatIdentity | undefined {
    if (!raw) return undefined

    let parsed: unknown
    try {
        parsed = JSON.parse(raw)
    } catch {
        return undefined
    }

    if (typeof parsed !== "object" || parsed === null) return undefined

    const {authorId, name} = parsed as {authorId?: unknown, name?: unknown}
    if (typeof authorId !== "string" || authorId === "") return undefined

    return {
        authorId,
        name: typeof name === "string" && isValidName(name) ? name.trim() : "",
    }
}

export function resolveIdentity(stored: string | null | undefined): ChatIdentity {
    return parseStoredIdentity(stored) ?? {authorId: generateUUID(), name: ""}
}

export function isValidName(name: string): boolean {
    const trimmed = name.trim()
    return trimmed !== "" && countRunes(trimmed) <= MAX_NAME_LENGTH
}
