import {Countries} from "./domain/countries.ts"

export const FLAG_CODES_SLOT = "__FLAG_CODES__"

// The home page's first script runs before any module, so the build writes the flags into it.
export function withFlagCodes(page: string): string {
    return page.replace(FLAG_CODES_SLOT, [...Countries.keys()].join(" "))
}
