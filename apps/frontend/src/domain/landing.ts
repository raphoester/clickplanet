export const SOURCE_TAG_PARAM = "s"

export const RECRUIT_TAG = "rc"

// What the first page said, for the backend to keep with the account it makes.
export type Landing = {
    tag: string
    referrer: string
}

export function landingOf(url: URL, referrer: string): Landing {
    return {
        tag: url.searchParams.get(SOURCE_TAG_PARAM)?.trim().toLowerCase() ?? "",
        referrer: hostOf(referrer, url.hostname),
    }
}

// The host alone: the path and the query of the page before may say who the player is.
function hostOf(referrer: string, own: string): string {
    try {
        const host = new URL(referrer).hostname.toLowerCase()
        return host === own ? "" : host
    } catch {
        return ""
    }
}

export function withoutLanding(url: URL): URL {
    const stripped = new URL(url)
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
