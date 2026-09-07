import {describe, expect, it} from "vitest"
import {truncate} from "./truncate.ts"
import {Countries} from "../domain/countries.ts"

describe("truncate", () => {
    it("leaves a short string alone", () => {
        expect(truncate("France", 18)).toBe("France")
    })

    it("marks a string it had to shorten", () => {
        expect(truncate("abcdefghij", 5)).toBe("abcde…")
    })

    it("does not mark a string that exactly fits", () => {
        expect(truncate("abcde", 5)).toBe("abcde")
    })

    it("does not leave a trailing space before the ellipsis", () => {
        expect(truncate("ab cdef", 3)).toBe("ab…")
    })

    /**
     * Every country name is prefixed with a flag. Counting UTF-16 units made
     * the emoji itself eat most of the budget: the UK nations' flags are tag
     * sequences fourteen units long, so England used to render as "Eng".
     */
    it("counts a flag emoji as one character", () => {
        expect(truncate("🏴󠁧󠁢󠁥󠁮󠁧󠁿 England", 18)).toBe("🏴󠁧󠁢󠁥󠁮󠁧󠁿 England")
        expect(truncate("🇬🇼 Guinea-Bissau", 18)).toBe("🇬🇼 Guinea-Bissau")
    })

    it("never splits an emoji in half", () => {
        const result = truncate("🇫🇷🇯🇵🇩🇪", 2)
        expect(result).toBe("🇫🇷🇯🇵…")
        expect(result).not.toContain("�")
    })

    it("keeps every real country name intact at the leaderboard's width", () => {
        const mangled = Array.from(Countries.values())
            .filter(c => truncate(c.name, 18) !== c.name)
            .map(c => c.name)
        expect(mangled).toEqual([])
    })
})
