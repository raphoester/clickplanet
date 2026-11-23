import {describe, expect, it} from "vitest"
import {looksLikeAnAddress} from "./emailAddress.ts"

describe("looksLikeAnAddress", () => {
    it.each(["ana@example.com", " ana@example.com ", "ana.b+game@mail.example.org"])("takes %j", (address) => {
        expect(looksLikeAnAddress(address)).toBe(true)
    })

    it.each(["", "ana", "ana@example", "@example.com", "ana@@example.com", "ana b@example.com"])("refuses %j", (address) => {
        expect(looksLikeAnAddress(address)).toBe(false)
    })
})
