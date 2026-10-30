import {SHARE_FLAG_PARAM, sharedCountry} from "./shareCard.ts"

export const SOURCE_TAG_PARAM = "s"

// The query of the first page of the game, for the backend to keep with the account it makes.
export type Landing = {
    flag: string
    tag: string
}

export function landingOf(url: URL): Landing {
    return {
        flag: sharedCountry(url)?.code ?? "",
        tag: url.searchParams.get(SOURCE_TAG_PARAM)?.trim().toLowerCase() ?? "",
    }
}

export function hasLanding(url: URL): boolean {
    return url.searchParams.has(SHARE_FLAG_PARAM) || url.searchParams.has(SOURCE_TAG_PARAM)
}

export function withoutLanding(url: URL): URL {
    const stripped = new URL(url)
    stripped.searchParams.delete(SHARE_FLAG_PARAM)
    stripped.searchParams.delete(SOURCE_TAG_PARAM)
    return stripped
}

export type LandingStore = {
    read(): Landing | undefined
    write(landing: Landing): void
}

// Kept from the first page on: the account is made at the first click or sign-in, often pages later.
export function firstLanding(store: LandingStore, seen: Landing): Landing {
    const kept = store.read()
    if (kept) return kept

    store.write(seen)
    return seen
}
