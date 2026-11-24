import {describe, expect, it} from "vitest"
import {FakeAdmin, keptSubmission, soldFlag} from "../fakeAdmin.ts"
import {Flag} from "../gen/grpc/flags/v1/flags_pb.ts"
import {foldName, founderOf, looksLikePhoto, nearOf, nextOf, PHOTO_FROM_PERCENT, sameNameOf, shownOf} from "./submissions.ts"

const {listed, submissions} = new FakeAdmin().flags
const byId = (id: string) => submissions.find(each => each.id === id)!

describe("the filters", () => {
    it("keeps the waiting ones, the approved new flags to open, or all", () => {
        expect(shownOf(submissions, "waiting").map(each => each.id)).toEqual(["sub-catalonia", "sub-beach", "sub-bavaria"])
        expect(shownOf(submissions, "toOpen").map(each => each.id)).toEqual(["sub-sardinia"])
        expect(shownOf(submissions, "all")).toHaveLength(6)
    })

    it("opens the next waiting one after a verdict, or the first one left", () => {
        const waiting = shownOf(submissions, "waiting")
        expect(nextOf(waiting, "sub-catalonia")).toBe("sub-beach")
        expect(nextOf(waiting, "sub-bavaria")).toBe("sub-catalonia")
        expect(nextOf(waiting.slice(0, 1), "sub-catalonia")).toBe("")
    })
})

describe("nearOf", () => {
    it("lists the other pins within 1,000 km, nearest first, but the flag a change is for", () => {
        const near = nearOf(byId("sub-bavaria"), listed, submissions)
        expect(near.map(each => [each.other.kind === "flag" ? each.other.flag.name : each.other.submission.name, Math.round(each.km)]))
            .toEqual([["Lake Garda", 292], ["Savoy", 516], ["Brittany", 982]])
    })

    it("leaves out a submission opened, whose flag is listed, and a change, whose flag is", () => {
        const near = nearOf(byId("sub-catalonia"), listed, submissions)
        const names = near.map(each => each.other.kind === "flag" ? `flag ${each.other.flag.name}` : `submission ${each.other.submission.name}`)
        expect(names).toEqual(["flag Savoy", "submission Sardinia", "submission Brittany", "submission Lake Garda"])
    })

    it("shows the nearest five at most", () => {
        const many = Array.from({length: 8}, (_, i) => soldFlag(`r-near00000${i}`, `Near ${i}`, new Date(), undefined, undefined,
            {latitude: 41.4 + i / 10, longitude: 2.17}))
        expect(nearOf(byId("sub-catalonia"), many, [])).toHaveLength(5)
    })
})

describe("sameNameOf", () => {
    it("finds a flag or another submission of the same name, case and accents aside", () => {
        const flags = [...listed, new Flag({code: "gb-brt", name: "Brittany"})]
        const others = [...submissions, keptSubmission("sub-savoie", "SAVÔY", {latitude: 45.5, longitude: 6})]

        const brittany = keptSubmission("sub-again", "brittany", {latitude: 48, longitude: -2})
        expect(sameNameOf(brittany, flags, others).map(other => other.kind === "flag" ? other.flag.code : other.submission.id))
            .toEqual(["gb-brt", "sub-brittany"])
        // Savoy's own submission opened its flag: the flag is listed once.
        expect(sameNameOf(byId("sub-savoy"), flags, others).map(other => other.kind === "flag" ? other.flag.code : other.submission.id))
            .toEqual(["sub-savoie"])
    })

    it("folds case and accents alone", () => {
        expect(foldName(" Île-de-France ")).toBe("ile-de-france")
        expect(foldName("Łódź")).toBe(foldName("lodz"))
        expect(foldName("Straße")).toBe("strasse")
        expect(foldName("Ile de France")).not.toBe(foldName("Île-de-France"))
    })
})

describe("a submission in words", () => {
    it("warns of a photo from PHOTO_FROM_PERCENT of other colours", () => {
        expect(looksLikePhoto(byId("sub-beach"))).toBe(true)
        expect(looksLikePhoto(keptSubmission("x", "x", {latitude: 0, longitude: 0}, {otherColorsPercent: PHOTO_FROM_PERCENT - 1}))).toBe(false)
        expect(looksLikePhoto(keptSubmission("x", "x", {latitude: 0, longitude: 0}, {otherColorsPercent: PHOTO_FROM_PERCENT}))).toBe(true)
    })

    it("names the founder, or its account's first group when no player has that id", () => {
        const names = new Map([["0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", "BraveFox42"]])
        expect(founderOf("0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10", names)).toBe("BraveFox42")
        expect(founderOf("9a1c0d2e-0000-4000-8000-000000000001", names)).toBe("9a1c0d2e…")
        expect(founderOf("9a1c0d2e-0000-4000-8000-000000000001", undefined)).toBe("9a1c0d2e…")
    })
})
