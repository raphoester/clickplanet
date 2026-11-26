import {describe, expect, it} from "vitest"
import {
    arrivalBarsOf, colorOf, everyOf, experimentsOf, labelled, lanesOf, logShareOf, logTicksOf, nearestOf, rightBarOf,
    upBarOf,
} from "./chart.ts"
import {Post} from "./readout.ts"

function post(experiment: string, platform: Post["platform"] = "tiktok"): Post {
    return {
        postId: `${experiment}-${platform}`, platform, experiment, publishedAt: new Date("2026-11-21T12:00:00Z"),
        day: "2026-11-21", ageHours: 48, url: "", views: 500, likes: 0, comments: 0, shares: 0, follows: 0, saves: 0,
    }
}

describe("logTicksOf", () => {
    it("ticks each power of ten from 100 to the first at or past the highest", () => {
        expect(logTicksOf(3000)).toEqual([100, 1000, 10_000])
        expect(logTicksOf(10_000)).toEqual([100, 1000, 10_000])
        expect(logTicksOf(10_001)).toEqual([100, 1000, 10_000, 100_000])
    })

    it("keeps two ticks when every value is small", () => {
        expect(logTicksOf(0)).toEqual([100, 1000])
    })
})

describe("logShareOf", () => {
    it("places a value on the log axis, and a value under the first tick at it", () => {
        const ticks = [100, 1000, 10_000]
        expect(logShareOf(1000, ticks)).toBe(0.5)
        expect(logShareOf(10_000, ticks)).toBe(1)
        expect(logShareOf(30, ticks)).toBe(0)
        expect(logShareOf(0, ticks)).toBe(0)
    })
})

describe("colorOf", () => {
    it("gives each experiment its slot by the sorted names, on both platforms, and the control grey", () => {
        const experiments = experimentsOf([
            post("war-report"), post("control"), post("carousel", "youtube"), post("far-invasion"), post("carousel"),
        ])

        expect(experiments).toEqual(["carousel", "far-invasion", "war-report"])
        expect(experiments.map(name => colorOf(name, experiments))).toEqual(["var(--lab-1)", "var(--lab-2)", "var(--lab-3)"])
        expect(colorOf("control", experiments)).toBe("var(--lab-control)")
    })

    it("draws an experiment past the sixth in ink, not a seventh hue", () => {
        const experiments = ["a", "b", "c", "d", "e", "f", "g"]
        expect(colorOf("f", experiments)).toBe("var(--lab-6)")
        expect(colorOf("g", experiments)).toBe("var(--text)")
    })
})

describe("lanesOf", () => {
    it("keeps dots in the middle lane until two come nearer than the gap", () => {
        expect(lanesOf([0, 20, 40], 10)).toEqual([0, 0, 0])
        expect(lanesOf([0, 4, 8, 12], 10)).toEqual([0, 1, 2, 0])
    })

    it("gives the lanes by the dots' places, whatever their order", () => {
        expect(lanesOf([12, 0, 8, 4], 10)).toEqual([0, 0, 2, 1])
    })

    it("puts a dot in the lane that ended first once all are taken", () => {
        expect(lanesOf([0, 1, 2, 3], 10)).toEqual([0, 1, 2, 0])
    })
})

describe("nearestOf", () => {
    const spots = [{key: "a", x: 10, y: 10}, {key: "b", x: 30, y: 10}]

    it("picks the spot nearest the pointer within reach", () => {
        expect(nearestOf(spots, 18, 12, 16)).toBe("a")
        expect(nearestOf(spots, 22, 12, 16)).toBe("b")
    })

    it("picks none past reach", () => {
        expect(nearestOf(spots, 60, 10, 16)).toBeUndefined()
        expect(nearestOf([], 0, 0, 16)).toBeUndefined()
    })
})

describe("labelled", () => {
    it("labels one day in every so many, the last day always", () => {
        const every = everyOf(18, 48)
        expect(every).toBe(3)
        expect(Array.from({length: 7}, (_, i) => labelled(i, 7, every))).toEqual([true, false, false, true, false, false, true])
        expect(everyOf(80, 48)).toBe(1)
    })
})

describe("arrivalBarsOf", () => {
    it("orders the days oldest first, today's still filling", () => {
        const bars = arrivalBarsOf([
            {day: "2026-11-22", tiktok: 1, youtube: 4},
            {day: "2026-11-21", tiktok: 0, youtube: 0},
            {day: "2026-11-20", tiktok: 2, youtube: 6},
        ], "2026-11-22")

        expect(bars).toEqual([
            {day: "2026-11-20", tiktok: 2, youtube: 6, filling: false},
            {day: "2026-11-21", tiktok: 0, youtube: 0, filling: false},
            {day: "2026-11-22", tiktok: 1, youtube: 4, filling: true},
        ])
    })
})

describe("bars", () => {
    it("rounds a bar at its end, square at its baseline", () => {
        expect(rightBarOf({x: 0, y: 10, length: 50, thickness: 12}))
            .toBe("M0,10 H46 A4,4 0 0 1 50,14 V18 A4,4 0 0 1 46,22 H0 Z")
        expect(upBarOf({x: 5, y: 100, length: 30, thickness: 10}))
            .toBe("M5,100 V74 A4,4 0 0 1 9,70 H11 A4,4 0 0 1 15,74 V100 Z")
    })

    it("rounds a short bar no further than its length", () => {
        expect(rightBarOf({x: 0, y: 0, length: 2, thickness: 12})).toBe("M0,0 H0 A2,2 0 0 1 2,2 V10 A2,2 0 0 1 0,12 H0 Z")
    })
})
