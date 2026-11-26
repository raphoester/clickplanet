import {describe, expect, it} from "vitest"
import {
    changeable,
    flagNameLength,
    foundedFlags,
    isValidFlagName,
    MAX_PICTURE_BYTES,
    onItsWayOf,
    pictureFileRefusal,
    pictureShapeRefusal,
    pinAt,
    shownOf,
    Submission,
    untold,
    withTold,
} from "./flagSubmissions.ts"
import {FlagCatalog} from "./flags.ts"

const submission = (id: string, changes: Partial<Submission> = {}): Submission => ({
    id, name: "Bavaria", flag: "", state: "waiting", submittedAt: 1, decidedAt: 1, paid: false, ...changes,
})

describe("a flag's name", () => {
    it("is 1 to 40 characters once the spaces at its ends are cut, counted as code points", () => {
        expect(isValidFlagName("  ")).toBe(false)
        expect(isValidFlagName(" Bavaria ")).toBe(true)
        expect(flagNameLength(" Bavaria ")).toBe(7)
        expect(isValidFlagName("a".repeat(40))).toBe(true)
        expect(isValidFlagName("a".repeat(41))).toBe(false)
        expect(flagNameLength("🏴".repeat(40))).toBe(40)
        expect(isValidFlagName("🏴".repeat(40))).toBe(true)
    })
})

describe("a flag's picture", () => {
    it("is a PNG, a JPEG or a WebP, not empty, at most 5 MB", () => {
        expect(pictureFileRefusal({type: "image/png", size: 10})).toBeUndefined()
        expect(pictureFileRefusal({type: "image/webp", size: MAX_PICTURE_BYTES})).toBeUndefined()
        expect(pictureFileRefusal({type: "image/gif", size: 10})).toBe("type")
        expect(pictureFileRefusal({type: "image/jpeg", size: 0})).toBe("type")
        expect(pictureFileRefusal({type: "image/jpeg", size: MAX_PICTURE_BYTES + 1})).toBe("size")
    })

    it("is between square and 2 wide by 1 high, and 100 pixels wide at least", () => {
        expect(pictureShapeRefusal(300, 200)).toBeUndefined()
        expect(pictureShapeRefusal(200, 200)).toBeUndefined()
        expect(pictureShapeRefusal(400, 200)).toBeUndefined()
        expect(pictureShapeRefusal(200, 201)).toBe("narrow")
        expect(pictureShapeRefusal(401, 200)).toBe("wide")
        expect(pictureShapeRefusal(99, 60)).toBe("small")
    })
})

describe("pinAt", () => {
    it("reads a point as the tile positions place it: y the north pole, longitude 0 on -x", () => {
        const close = (pin: {latitude: number, longitude: number}, latitude: number, longitude: number) => {
            expect(pin.latitude).toBeCloseTo(latitude, 6)
            expect(pin.longitude).toBeCloseTo(longitude, 6)
        }
        close(pinAt({x: -2, y: 0, z: 0}), 0, 0)
        close(pinAt({x: 0, y: 0, z: 3}), 0, 90)
        close(pinAt({x: 0, y: 0, z: -1}), 0, -90)
        expect(pinAt({x: 0, y: 5, z: 0}).latitude).toBeCloseTo(90, 6)
        const phi = 48 * Math.PI / 180, lambda = 11.5 * Math.PI / 180
        close(pinAt({x: -Math.cos(phi) * Math.cos(lambda), y: Math.sin(phi), z: Math.cos(phi) * Math.sin(lambda)}), 48, 11.5)
    })
})

describe("the account's submissions", () => {
    it("hold the new flag on its way, waiting or approved and not open yet, the newest first", () => {
        expect(onItsWayOf([submission("a", {state: "rejected"}), submission("o", {state: "opened", flag: "r-aaaaaaaaaa"})])).toBeUndefined()
        expect(onItsWayOf([submission("b"), submission("a", {state: "rejected"})])?.id).toBe("b")
        expect(onItsWayOf([submission("c", {state: "rejected"}), submission("b", {state: "approved", paid: true})])?.id).toBe("b")
    })

    it("change while it waits and is not paid, never once paid or decided", () => {
        expect(changeable(submission("a"))).toBe(true)
        expect(changeable(submission("a", {paid: true}))).toBe(false)
        expect(changeable(submission("a", {state: "approved"}))).toBe(false)
        expect(changeable(submission("a", {state: "rejected"}))).toBe(false)
    })

    it("name the flags the account founded, the newest first: each opened from one of them", () => {
        expect(foundedFlags([
            submission("c", {state: "waiting"}),
            submission("b", {flag: "r-bbbbbbbbbb", state: "opened"}),
            submission("a", {flag: "r-aaaaaaaaaa", state: "opened"}),
            submission("x", {state: "approved"}),
        ])).toEqual(["r-bbbbbbbbbb", "r-aaaaaaaaaa"])
    })

    it("tell each decision once, and a submission approved, opened, then its flag closed thrice", () => {
        const approved = [submission("b", {state: "approved"}), submission("a", {state: "rejected"}), submission("w")]
        expect(untold(approved, {}, FlagCatalog.BUILT_IN).map(({id}) => id)).toEqual(["b", "a"])

        const told = withTold({}, untold(approved, {}, FlagCatalog.BUILT_IN))
        expect(untold(approved, told, FlagCatalog.BUILT_IN)).toEqual([])

        const opened = [submission("b", {state: "opened", flag: "r-aaaaaaaaaa"}), ...approved.slice(1)]
        expect(untold(opened, told, catalogOf(false)).map(({id, state}) => [id, state])).toEqual([["b", "opened"]])

        const toldOpened = withTold(told, untold(opened, told, catalogOf(false)))
        expect(untold(opened, toldOpened, catalogOf(true)).map(({id, state}) => [id, state])).toEqual([["b", "closed"]])
    })
})

const catalogOf = (closed: boolean) => FlagCatalog.BUILT_IN.withFlags([{code: "r-aaaaaaaaaa", name: "Bavaria", closed}])

describe("a submission as its player reads it", () => {
    it("is its flag's state now once it opened the flag, and its own before", () => {
        const opened = submission("a", {state: "opened", flag: "r-aaaaaaaaaa"})
        expect(shownOf(opened, catalogOf(false)).state).toBe("opened")
        expect(shownOf(opened, catalogOf(true)).state).toBe("closed")

        expect(shownOf(submission("b", {state: "approved"}), catalogOf(true)).state).toBe("approved")
    })
})
