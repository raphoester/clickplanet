import {readdirSync, readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {COMPACT, LANDSCAPE} from "./compact.ts"

const SRC = fileURLToPath(new URL("..", import.meta.url))

const queriesMatching = (pattern: RegExp) => readdirSync(SRC, {recursive: true, encoding: "utf8"})
    .filter((file) => file.endsWith(".css"))
    .flatMap((file) => [...readFileSync(SRC + file, "utf8").matchAll(pattern)]
        .map((match) => `${file}: ${match[1].trim()}`))

describe("the compact layout", () => {
    it("is the same query in every stylesheet as in useCompact", () => {
        const queries = queriesMatching(/@media ([^{]*768px[^{]*)\{/g)

        expect(queries.length).toBeGreaterThan(0)
        expect(queries.filter((query) => !query.endsWith(`: ${COMPACT}`))).toEqual([])
    })

    it("held sideways is the same query in every stylesheet as in useLandscape", () => {
        const queries = queriesMatching(/@media ([^{]*landscape[^{]*)\{/g)

        expect(queries.length).toBeGreaterThan(0)
        expect(queries.filter((query) => !query.endsWith(`: ${LANDSCAPE}`))).toEqual([])
    })

    it("held sideways is a compact screen", () => {
        expect(LANDSCAPE).toContain("(max-height: 600px)")
        expect(COMPACT.split(", ")).toContain("(max-height: 600px)")
    })
})
