export const SOURCE_TAG_PARAM = "s"

export const RECRUIT_TAG = "rc"

export const RECRUITER_PARAM = "r"

// What the first page said, for the backend to keep with the account it makes.
export type Landing = {
    tag: string
    referrer: string
    recruiterAccountId?: string
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

// A username holds no "-": the address writes each of its spaces as one.
export function recruiterNameOf(url: URL): string | undefined {
    const name = url.searchParams.get(RECRUITER_PARAM)?.replaceAll("-", " ")
    return name ? name : undefined
}

export function withoutLanding(url: URL): URL {
    const stripped = new URL(url)
    stripped.searchParams.delete(SOURCE_TAG_PARAM)
    stripped.searchParams.delete(RECRUITER_PARAM)
    return stripped
}

export type LandingStore = {
    read(): Landing | undefined
    write(landing: Landing): void
}

export type Recruiters = (name: string) => Promise<string | undefined>

// Kept from the first page on: the account is made at the first click or sign-in, often pages later.
export async function firstLanding(
    store: LandingStore,
    seen: Landing,
    recruiter?: {name: string, accountOf: Recruiters},
): Promise<Landing> {
    const kept = store.read()
    if (kept) return kept

    store.write(seen)
    if (!recruiter) return seen

    // The landing goes with the call that makes the account: a failed read must not fail it.
    const recruiterAccountId = await recruiter.accountOf(recruiter.name).catch(() => undefined)
    if (!recruiterAccountId) return seen

    const recruited = {...seen, recruiterAccountId}
    store.write(recruited)
    return recruited
}
