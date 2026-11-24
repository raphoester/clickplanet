import {describe, expect, it} from "vitest"
import {nameRefusalOf} from "./fakeAdmin.ts"

describe("the fake's flag names", () => {
    it("refuses the name of a flag the game ships with, whatever its case, as the server does", () => {
        expect(nameRefusalOf("brittany", [])).toBe("a flag the game ships with is named Brittany")
        expect(nameRefusalOf("FRANCE", [])).toBe("a flag the game ships with is named France")
        expect(nameRefusalOf("Bayern", [])).toBeUndefined()
    })
})
