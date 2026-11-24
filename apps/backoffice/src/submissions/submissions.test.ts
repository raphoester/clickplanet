import {describe, expect, it} from "vitest"
import {FakeAdmin, keptSubmission, soldFlag} from "../fakeAdmin.ts"
import {JudgementVerdict} from "../gen/grpc/flags/v1/admin_pb.ts"
import {Flag} from "../gen/grpc/flags/v1/flags_pb.ts"
import {
    aroundOf, foldName, founderOf, judgementLabel, looksLikePhoto, Near, nearOf, nextOf, Other, PHOTO_FROM_PERCENT,
    sameNameOf, shownOf,
} from "./submissions.ts"

const {listed, submissions} = new FakeAdmin().flags
const byId = (id: string) => submissions.find(each => each.id === id)!
const MADRID = {country: "es", latitude: 40.4168, longitude: -3.7038}
const ANDORRA = {country: "ad", latitude: 42.5063, longitude: 1.5218}
const PARIS = {country: "fr", latitude: 48.8566, longitude: 2.3522}
const TOKYO = {country: "jp", latitude: 35.6762, longitude: 139.6503}

function nameOf(other: Other): string {
    switch (other.kind) {
        case "flag":
            return `flag ${other.flag.name}`
        case "submission":
            return `submission ${other.submission.name}`
        case "country":
            return `country ${other.country}`
    }
}

const namesOf = (near: Near[]) => near.map(each => [nameOf(each.other), Math.round(each.km)])
const pinOf = (id: string) => ({latitude: byId(id).pin!.latitude, longitude: byId(id).pin!.longitude})

describe("the filters", () => {
    it("keeps the waiting ones, the approved new flags to open, the ones whose founder replied to their letter, or all", () => {
        expect(shownOf(submissions, "waiting", new Set()).map(each => each.id)).toEqual(["sub-catalonia", "sub-beach", "sub-bavaria"])
        expect(shownOf(submissions, "toOpen", new Set()).map(each => each.id)).toEqual(["sub-sardinia"])
        expect(shownOf(submissions, "replies", new Set(["sub-catalonia"])).map(each => each.id)).toEqual(["sub-catalonia"])
        expect(shownOf(submissions, "all", new Set())).toHaveLength(6)
    })

    it("opens the next waiting one after a verdict, or the first one left", () => {
        const waiting = shownOf(submissions, "waiting", new Set())
        expect(nextOf(waiting, "sub-catalonia")).toBe("sub-beach")
        expect(nextOf(waiting, "sub-bavaria")).toBe("sub-catalonia")
        expect(nextOf(waiting.slice(0, 1), "sub-catalonia")).toBe("")
    })
})

describe("nearOf", () => {
    it("lists the other capitals within 1,000 km, nearest first", () => {
        const near = nearOf(pinOf("sub-bavaria"), aroundOf(byId("sub-bavaria"), listed, submissions, []))
        expect(namesOf(near)).toEqual([
            ["flag Bavaria", 0], ["submission Lake Garda", 292], ["flag Savoy", 516], ["submission Brittany", 982],
        ])
    })

    it("leaves out a submission opened, whose flag is listed", () => {
        const near = nearOf(pinOf("sub-catalonia"), aroundOf(byId("sub-catalonia"), listed, submissions, []))
        expect(near.map(each => nameOf(each.other))).toEqual(["flag Savoy", "submission Sardinia", "submission Brittany", "submission Lake Garda"])
    })

    it("adds the game's countries' capitals, with their distance", () => {
        const around = aroundOf(byId("sub-catalonia"), listed, submissions, [MADRID, ANDORRA, PARIS, TOKYO])
        expect(namesOf(nearOf(pinOf("sub-catalonia"), around)).slice(0, 4))
            .toEqual([["country ad", 135], ["country es", 505], ["flag Savoy", 554], ["submission Sardinia", 637]])
        expect(around.map(each => nameOf(each.other))).toContain("country jp")
        expect(namesOf(nearOf(pinOf("sub-catalonia"), around)).map(([name]) => name)).not.toContain("country jp")
    })

    it("measures from where the capital moved", () => {
        const around = aroundOf(byId("sub-catalonia"), listed, submissions, [MADRID, ANDORRA, PARIS])
        expect(namesOf(nearOf({latitude: 48.85, longitude: 2.35}, around))[0]).toEqual(["country fr", 1])
    })

    it("shows the nearest five flags and submissions, and the nearest five countries", () => {
        const many = Array.from({length: 8}, (_, i) => soldFlag(`r-near00000${i}`, `Near ${i}`, new Date(), undefined, undefined,
            {latitude: 41.4 + i / 10, longitude: 2.17}))
        const countries = Array.from({length: 8}, (_, i) => ({country: `c${i}`, latitude: 41 - i / 10, longitude: 2.17}))
        const near = nearOf(pinOf("sub-catalonia"), aroundOf(byId("sub-catalonia"), many, [], countries))
        expect(near.filter(each => each.other.kind === "flag")).toHaveLength(5)
        expect(near.filter(each => each.other.kind === "country")).toHaveLength(5)
    })
})

describe("sameNameOf", () => {
    it("finds a flag or another submission of the same name, case and accents aside", () => {
        const flags = [...listed, new Flag({code: "gb-brt", name: "Brittany"})]
        const others = [...submissions, keptSubmission("sub-savoie", "SAVÔY", {latitude: 45.5, longitude: 6})]

        const brittany = keptSubmission("sub-again", "brittany", {latitude: 48, longitude: -2})
        expect(sameNameOf(brittany, flags, others).map(nameOf)).toEqual(["flag Brittany", "submission Brittany"])
        // Savoy's own submission opened its flag: the flag is listed once.
        expect(sameNameOf(byId("sub-savoy"), flags, others).map(nameOf)).toEqual(["submission SAVÔY"])
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

describe("judgementLabel", () => {
    it("says what the judge ruled, and that an unsure one is the operator's", () => {
        expect(judgementLabel(JudgementVerdict.APPROVE)).toBe("Approved")
        expect(judgementLabel(JudgementVerdict.REJECT)).toBe("Rejected")
        expect(judgementLabel(JudgementVerdict.UNSURE)).toBe("Unsure, left to you")
    })
})
