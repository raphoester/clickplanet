import {describe, expect, it} from "vitest"
import {truncate} from "./truncate.ts"
import {Countries} from "../domain/countries.ts"
import {translator} from "../i18n/page.ts"

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

    it("counts a flag emoji as one character", () => {
        expect(truncate("🏴󠁧󠁢󠁥󠁮󠁧󠁿 England", 18)).toBe("🏴󠁧󠁢󠁥󠁮󠁧󠁿 England")
        expect(truncate("🇬🇼 Guinea-Bissau", 18)).toBe("🇬🇼 Guinea-Bissau")
    })

    it("never splits an emoji in half", () => {
        const result = truncate("🇫🇷🇯🇵🇩🇪", 2)
        expect(result).toBe("🇫🇷🇯🇵…")
        expect(result).not.toContain("�")
    })

    it("keeps a real country name whole at the leaderboard's width, or its start and the ellipsis", () => {
        const mangled = Array.from(Countries.keys())
            .map(code => translator().country(code))
            .filter(name => {
                const cut = truncate(name, 18)
                return cut !== name && !(cut.endsWith("…") && name.startsWith(cut.slice(0, -1)))
            })
        expect(mangled).toEqual([])
    })
})
