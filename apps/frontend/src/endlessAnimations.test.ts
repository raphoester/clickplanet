import {readdirSync, readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"

const SRC = fileURLToPath(new URL(".", import.meta.url))

// What a player waits on: paused, it would look stuck.
const WAITS = new Set(["sign-in-callback-spin", "viewer-status-spin", "battle-over-wait", "quiz-picked"])

// Shown only while the player aims, or waits on the server.
const REPAINTS = new Set(["inventory-loop", "quiz-picked"])

const COMPOSITED = new Set(["opacity", "transform", "scale", "rotate", "translate"])

type Endless = {file: string, name: string, item: string}

function stylesheets(): {file: string, css: string}[] {
    return readdirSync(SRC, {recursive: true, encoding: "utf8"})
        .filter((file) => file.endsWith(".css"))
        .map((file) => ({file, css: readFileSync(SRC + file, "utf8").replace(/\/\*[\s\S]*?\*\//g, "")}))
}

function topLevelItems(value: string): string[] {
    const items: string[] = []
    let depth = 0
    let start = 0
    for (let i = 0; i < value.length; i++) {
        if (value[i] === "(") depth++
        if (value[i] === ")") depth--
        if (value[i] === "," && depth === 0) {
            items.push(value.slice(start, i).trim())
            start = i + 1
        }
    }
    return [...items, value.slice(start).trim()]
}

function keyframesOf(css: string): Map<string, Set<string>> {
    const found = new Map<string, Set<string>>()
    for (const match of css.matchAll(/@keyframes\s+([\w-]+)\s*\{/g)) {
        let depth = 1
        let end = match.index + match[0].length
        while (depth > 0 && end < css.length) {
            if (css[end] === "{") depth++
            if (css[end] === "}") depth--
            end++
        }
        const body = css.slice(match.index + match[0].length, end - 1)
        const properties = [...body.matchAll(/([\w-]+)\s*:/g)].map((property) => property[1])
        found.set(match[1], new Set(properties))
    }
    return found
}

function endlessAnimations(names: Set<string>): Endless[] {
    return stylesheets().flatMap(({file, css}) =>
        [...css.matchAll(/(?:^|[;{\s])animation\s*:\s*([^;}]+)/g)].flatMap((declaration) =>
            topLevelItems(declaration[1])
                .filter((item) => /\binfinite\b/.test(item))
                .map((item) => ({file, item, name: item.split(/\s+/).find((word) => names.has(word)) ?? "?"}))))
}

describe("an endless animation", () => {
    const keyframes = new Map(stylesheets().flatMap(({css}) => [...keyframesOf(css)]))
    const endless = endlessAnimations(new Set(keyframes.keys()))

    it("is found in the stylesheets", () => {
        expect(endless.map(({name}) => name)).toContain("inventory-new")
        expect(endless.map(({name}) => name)).not.toContain("?")
    })

    it("pauses on a page nobody watches, but for what a player waits on", () => {
        const playing = endless.filter(({name, item}) => !WAITS.has(name) && !item.includes("var(--endless, running)"))

        expect(playing.map(({file, name}) => `${file}: ${name}`)).toEqual([])
    })

    it("moves only what the compositor plays on its own", () => {
        const repainting = endless
            .filter(({name}) => !REPAINTS.has(name))
            .flatMap(({file, name}) => [...keyframes.get(name) ?? []]
                .filter((property) => !COMPOSITED.has(property))
                .map((property) => `${file}: ${name} animates ${property}`))

        expect(repainting).toEqual([])
    })
})
