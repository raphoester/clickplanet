import {v4 as generateUUID} from 'uuid';

export const CHAT_IDENTITY_STORAGE_KEY = 'clickplanet-chat-identity'

/**
 * The id this browser posts under. It names no one — the server names the
 * sender by the click token — and is kept only because the request still
 * carries it. A name stored by an older build is ignored.
 */
export type ChatIdentity = {
    authorId: string
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

    const {authorId} = parsed as {authorId?: unknown}
    if (typeof authorId !== "string" || authorId === "") return undefined

    return {authorId}
}

export function resolveIdentity(stored: string | null | undefined): ChatIdentity {
    return parseStoredIdentity(stored) ?? {authorId: generateUUID()}
}
