import {readdirSync, readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {COMPACT} from "./compact.ts"

const SRC = fileURLToPath(new URL("..", import.meta.url))

describe("the compact layout", () => {
    it("is the same query in every stylesheet as in useCompact", () => {
        const queries = readdirSync(SRC, {recursive: true, encoding: "utf8"})
            .filter((file) => file.endsWith(".css"))
            .flatMap((file) => [...readFileSync(SRC + file, "utf8").matchAll(/@media ([^{]*768px[^{]*)\{/g)]
                .map((match) => `${file}: ${match[1].trim()}`))

        expect(queries.length).toBeGreaterThan(0)
        expect(queries.filter((query) => !query.endsWith(`: ${COMPACT}`))).toEqual([])
    })
})
