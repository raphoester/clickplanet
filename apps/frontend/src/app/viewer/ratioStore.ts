import {MAX_PIXEL_RATIO} from "./scene.ts"

export const RATIO_STORAGE_KEY = "clickplanet-pixel-ratio"

export type RatioStore = Pick<Storage, "getItem" | "setItem">

export const browserStore = (): RatioStore => window.localStorage

export function savedRatio(storeOf: () => RatioStore = browserStore): number {
    try {
        const saved = Number(storeOf().getItem(RATIO_STORAGE_KEY) ?? MAX_PIXEL_RATIO)
        return saved >= 1 && saved <= MAX_PIXEL_RATIO ? saved : MAX_PIXEL_RATIO
    } catch {
        return MAX_PIXEL_RATIO
    }
}

export function saveRatio(ratio: number, storeOf: () => RatioStore = browserStore) {
    try {
        storeOf().setItem(RATIO_STORAGE_KEY, String(ratio))
    } catch (e) {
        console.error("Could not keep the lowered pixel ratio", e)
    }
}
