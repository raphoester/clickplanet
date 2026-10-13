import {describe, expect, it} from "vitest"
import {ChatMessage} from "../backends/chat.ts"
import {NameColor} from "../backends/player.ts"
import {messageParts, textParts} from "./chatLinks.ts"

const message = (authorName: string, text: string): ChatMessage =>
    ({id: "1", sentAt: 0, authorAccountId: "", authorName, authorAdmin: false, authorColor: NameColor.UNSPECIFIED, authorStreak: 0, countryCode: "fr", text, reactions: [], reactionsVersion: 0})

describe("textParts", () => {
    it("leaves a text with no address as one part", () => {
        expect(textParts("who keeps taking Brittany")).toEqual([{kind: "text", text: "who keeps taking Brittany"}])
    })

    it("cuts an address out of the words around it", () => {
        expect(textParts("clips on https://www.tiktok.com/@clickplanetlol every day")).toEqual([
            {kind: "text", text: "clips on "},
            {kind: "link", text: "https://www.tiktok.com/@clickplanetlol", href: "https://www.tiktok.com/@clickplanetlol"},
            {kind: "text", text: " every day"},
        ])
    })

    it("finds every address of a text", () => {
        expect(textParts("http://a.example and https://b.example/x?y=1#z")).toEqual([
            {kind: "link", text: "http://a.example", href: "http://a.example/"},
            {kind: "text", text: " and "},
            {kind: "link", text: "https://b.example/x?y=1#z", href: "https://b.example/x?y=1#z"},
        ])
    })

    it("leaves the punctuation that ends a sentence out of the address", () => {
        expect(textParts("go to https://clickplanet.lol/play. Now!")).toEqual([
            {kind: "text", text: "go to "},
            {kind: "link", text: "https://clickplanet.lol/play", href: "https://clickplanet.lol/play"},
            {kind: "text", text: ". Now!"},
        ])
        expect(textParts("(https://clickplanet.lol)")).toEqual([
            {kind: "text", text: "("},
            {kind: "link", text: "https://clickplanet.lol", href: "https://clickplanet.lol/"},
            {kind: "text", text: ")"},
        ])
    })

    it("keeps a bracket the address opened", () => {
        const address = "https://en.wikipedia.org/wiki/Brittany_(administrative_region)"

        expect(textParts(`(${address})`)).toEqual([
            {kind: "text", text: "("},
            {kind: "link", text: address, href: address},
            {kind: "text", text: ")"},
        ])
    })

    it("links nothing but http and https", () => {
        for (const text of ["clickplanet.lol", "www.clickplanet.lol", "javascript:alert(1)", "ftp://a.example", "mailto:a@b.example", "https://", "https://..."]) {
            expect(textParts(text)).toEqual([{kind: "text", text}])
        }
    })

    it("does not link an address that names one site and goes to another", () => {
        const text = "free bombs https://clickplanet.lol@evil.example/login"

        expect(textParts(text)).toEqual([{kind: "text", text}])
    })
})

describe("messageParts", () => {
    it("links the addresses a player with a username wrote", () => {
        expect(messageParts(message("Ana", "see https://clickplanet.lol"))).toEqual([
            {kind: "text", text: "see "},
            {kind: "link", text: "https://clickplanet.lol", href: "https://clickplanet.lol/"},
        ])
    })

    it("leaves a guest's addresses as text", () => {
        expect(messageParts(message("guest_a1b2c3", "see https://clickplanet.lol"))).toEqual([
            {kind: "text", text: "see https://clickplanet.lol"},
        ])
    })
})
