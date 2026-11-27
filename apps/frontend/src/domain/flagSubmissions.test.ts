import {describe, expect, it} from "vitest"
import {
    changeable,
    flagRowsOf,
    flagNameLength,
    isValidFlagName,
    MAX_PICTURE_BYTES,
    pictureFileRefusal,
    pictureShapeRefusal,
    pinAt,
    pointOfPin,
    replacedByNewOf,
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

describe("pointOfPin", () => {
    it("puts a pin back on the sphere where pinAt read it", () => {
        for (const pin of [{latitude: 48.14, longitude: 11.58}, {latitude: -33.9, longitude: 151.2}, {latitude: 0, longitude: -179}]) {
            const back = pinAt(pointOfPin(pin))
            expect(back.latitude).toBeCloseTo(pin.latitude, 9)
            expect(back.longitude).toBeCloseTo(pin.longitude, 9)
        }
        expect(pointOfPin({latitude: 90, longitude: 0}).y).toBeCloseTo(1, 12)
    })
})

describe("the account's submissions", () => {
    it("give a new flag the place of the one that waits unpaid, never of one paid or decided", () => {
        expect(replacedByNewOf([submission("b"), submission("a", {state: "rejected"})])?.id).toBe("b")
        expect(replacedByNewOf([submission("b", {paid: true})])).toBeUndefined()
        expect(replacedByNewOf([submission("b", {state: "approved"}), submission("a", {state: "rejected"})])).toBeUndefined()
    })

    it("change while it waits and is not paid, never once paid or decided", () => {
        expect(changeable(submission("a"))).toBe(true)
        expect(changeable(submission("a", {paid: true}))).toBe(false)
        expect(changeable(submission("a", {state: "approved"}))).toBe(false)
        expect(changeable(submission("a", {state: "rejected"}))).toBe(false)
    })

    it("tell each decision once, and a submission approved, opened, then its flag closed thrice", () => {
        const approved = [submission("b", {state: "approved"}), submission("a", {state: "rejected"}), submission("w")]
        const held = [{code: "r-aaaaaaaaaa"}]
        expect(untold(approved, {}, FlagCatalog.BUILT_IN, held).map(({id}) => id)).toEqual(["b", "a"])

        const told = withTold({}, untold(approved, {}, FlagCatalog.BUILT_IN, held))
        expect(untold(approved, told, FlagCatalog.BUILT_IN, held)).toEqual([])

        const opened = [submission("b", {state: "opened", flag: "r-aaaaaaaaaa"}), ...approved.slice(1)]
        expect(untold(opened, told, catalogOf(false), held).map(({id, state}) => [id, state])).toEqual([["b", "opened"]])

        const toldOpened = withTold(told, untold(opened, told, catalogOf(false), held))
        expect(untold(opened, toldOpened, catalogOf(true), held).map(({id, state}) => [id, state])).toEqual([["b", "closed"]])
    })

    it("tell nothing of a flag another player claimed since", () => {
        const opened = [submission("b", {state: "opened", flag: "r-aaaaaaaaaa"})]
        const told = withTold({}, untold(opened, {}, catalogOf(true), [{code: "r-aaaaaaaaaa"}]))

        expect(untold(opened, told, catalogOf(false), [])).toEqual([])
    })
})

describe("the rows of the account's flags", () => {
    const removed = FlagCatalog.BUILT_IN.withFlags([{code: "r-aaaaaaaaaa", name: "Bavaria", closed: true, sold: {
        picture: {url: "https://media.test/b.png", width: 768, height: 512, stretch: false, focusX: 0.5, focusY: 0.5},
        endsAt: 1, closedAt: 1, removed: true,
    }}])
    const keys = (rows: ReturnType<typeof flagRowsOf>) => rows.map(({key}) => key)

    it("are each flag held, then the newest flag sent of each name the account does not hold", () => {
        const mine = {
            held: [{code: "r-aaaaaaaaaa"}],
            submissions: [
                submission("c", {name: "swabia", state: "rejected", rejection: {reason: "exists", note: ""}}),
                submission("b", {name: "Swabia", state: "rejected", rejection: {reason: "other", note: "No."}}),
                submission("a", {name: "BAVARIA", state: "rejected", rejection: {reason: "exists", note: ""}}),
                submission("o", {state: "opened", flag: "r-aaaaaaaaaa"}),
            ],
        }
        expect(keys(flagRowsOf(mine, removed, new Set()))).toEqual(["flag:r-aaaaaaaaaa", "c"])
    })

    it("leave out what the player put away while it is done, and never what waits", () => {
        const mine = {
            held: [{code: "r-aaaaaaaaaa"}],
            submissions: [submission("w", {name: "Swabia"}), submission("p", {name: "Tyrol", state: "approved"})],
        }
        const dismissed = new Set(["flag:r-aaaaaaaaaa", "w", "p"])

        expect(keys(flagRowsOf(mine, removed, dismissed))).toEqual(["w"])
        expect(keys(flagRowsOf(mine, catalogOf(false), dismissed)), "open again, it is no longer done")
            .toEqual(["flag:r-aaaaaaaaaa", "w"])
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
