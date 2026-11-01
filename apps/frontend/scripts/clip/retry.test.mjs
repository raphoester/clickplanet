import {describe, expect, it} from "vitest"
import {withRetries} from "./retry.mjs"

const failing = (failures, answer) => {
    const tried = []
    return {
        tried,
        start: async (attempt) => {
            tried.push(attempt)
            if (tried.length <= failures) throw new Error("Chrome never exposed a debugging target")
            return answer
        },
    }
}

describe("a launch tried again", () => {
    it("answers at once when it starts, and logs nothing", async () => {
        const {start, tried} = failing(0, "ws://chrome")
        const logged = []

        expect(await withRetries(start, {tries: 3, log: (line) => logged.push(line)})).toBe("ws://chrome")
        expect(tried).toEqual([1])
        expect(logged).toEqual([])
    })

    it("is tried again after a failure, with a line that says so", async () => {
        const {start, tried} = failing(2, "ws://chrome")
        const logged = []

        expect(await withRetries(start, {tries: 3, log: (line) => logged.push(line)})).toBe("ws://chrome")
        expect(tried).toEqual([1, 2, 3])
        expect(logged).toEqual([
            "Chrome never exposed a debugging target: try 2 of 3",
            "Chrome never exposed a debugging target: try 3 of 3",
        ])
    })

    it("fails with the last error once every try failed", async () => {
        const {start, tried} = failing(3, "ws://chrome")

        await expect(withRetries(start, {tries: 3, log: () => {}})).rejects.toThrow("Chrome never exposed a debugging target")
        expect(tried).toEqual([1, 2, 3])
    })
})
