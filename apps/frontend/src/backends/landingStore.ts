import {Landing, LandingStore} from "../domain/landing.ts"

export const LANDING_STORAGE_KEY = "clickplanet-landing"

export function localLandingStore(): LandingStore {
    return {
        read(): Landing | undefined {
            try {
                const kept = window.localStorage.getItem(LANDING_STORAGE_KEY)
                if (!kept) return undefined

                const landing = JSON.parse(kept) as Partial<Landing>
                if (typeof landing.tag !== "string" || typeof landing.referrer !== "string") return undefined

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
