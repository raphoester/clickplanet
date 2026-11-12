import {Landing, LandingStore} from "../domain/landing.ts"

export const LANDING_STORAGE_KEY = "clickplanet-landing"

// The page before a flag's address, which the home page's first script hands to the game before it moves there.
export const HANDED_REFERRER_KEY = "clickplanet-referrer"

export function localLandingStore(): LandingStore {
    return {
        read(): Landing | undefined {
            try {
                const kept = window.localStorage.getItem(LANDING_STORAGE_KEY)
                if (!kept) return undefined

                const landing = JSON.parse(kept) as Partial<Landing>
                if (typeof landing.tag !== "string" || typeof landing.referrer !== "string") return undefined
                if (typeof landing.recruiterAccountId === "string") {
                    return {tag: landing.tag, referrer: landing.referrer, recruiterAccountId: landing.recruiterAccountId}
                }

                return {tag: landing.tag, referrer: landing.referrer}
            } catch {
                return undefined
            }
        },
        write(landing: Landing): void {
            try {
                window.localStorage.setItem(LANDING_STORAGE_KEY, JSON.stringify(landing))
            } catch {
                // storage unavailable
            }
        },
    }
}

// Read once: a reload of the game is not the page that came by the address.
export function handedReferrer(): string | undefined {
    try {
        const referrer = window.sessionStorage.getItem(HANDED_REFERRER_KEY)
        window.sessionStorage.removeItem(HANDED_REFERRER_KEY)
        return referrer ?? undefined
    } catch {
        return undefined
    }
}
