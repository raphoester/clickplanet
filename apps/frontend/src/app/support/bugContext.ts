import {BugContext} from "../../backends/support.ts"

export function buildOf(moduleUrl: string): string {
    try {
        return new URL(moduleUrl).pathname.split("/").pop() ?? ""
    } catch {
        return ""
    }
}

// The file the bundler put this module in: its name carries the build's hash.
export const BUILD = buildOf(import.meta.url)

export type Page = {
    innerWidth: number
    innerHeight: number
}

export function bugContextOf(page: Page, connectionDown: boolean): BugContext {
    return {
        build: BUILD,
        viewportWidth: Math.round(page.innerWidth),
        viewportHeight: Math.round(page.innerHeight),
        connectionDown,
    }
}
