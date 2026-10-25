import {describe, expect, it} from "vitest"
import {metricsAt} from "../fakeAdmin.ts"
import {Metric} from "../gen/grpc/metrics/v1/admin_pb.ts"
import {windowOf} from "../home/ledger.ts"
import {
    increaseOf,
    keptSamples,
    lastMinuteOf,
    MINUTE,
    named,
    nowOf,
    perSlotOf,
    reasonsOf,
    refused,
    Sample,
    sampleOf,
    shareUnderOf,
    sinceOf,
} from "./counters.ts"

function sample(at: number, value: number, startedAt = 0): Sample {
    return {at, startedAt, metrics: [new Metric({name: "throttled_clicks", value: {case: "counter", value}})]}
}

const throttled = named("throttled_clicks")

describe("the counters of the process", () => {
    it("reads when the server read them and sums the series a select picks, labels and all", () => {
        const read = sampleOf(metricsAt(0, 60))

        expect(read.at).toBe(new Date("2026-10-18T12:00:00Z").getTime())
        expect(nowOf([read], named("clicks_total"))).toBe(24_000)
        expect(nowOf([read], named("clicks_total", {country_id: "dz"}))).toBe(6_000)
        expect(nowOf([read], refused)).toBe(1_800 + 180)
    })

    it("counts what a counter counted between two reads, and all of it after a restart", () => {
        expect(increaseOf(sample(0, 10), sample(MINUTE, 40), throttled)).toBe(30)
        expect(increaseOf(sample(0, 10), sample(MINUTE, 4, 1), throttled)).toBe(4)
    })

    it("keeps the span and one sample before it, and a sample read twice once", () => {
        const samples = [sample(0, 1), sample(MINUTE, 2), sample(2 * MINUTE, 3)]

        expect(keptSamples(samples, sample(4 * MINUTE, 4), 2 * MINUTE).map(kept => kept.at)).toEqual([MINUTE, 2 * MINUTE, 4 * MINUTE])
        expect(keptSamples(samples, sample(2 * MINUTE, 3), 10 * MINUTE)).toHaveLength(3)
    })

    it("spreads what two reads counted over the minutes between them, and knows nothing of a minute none spans", () => {
        const window = windowOf(new Date(4 * MINUTE), 4 / 60, MINUTE)
        const samples = [sample(MINUTE + MINUTE / 2, 0), sample(2 * MINUTE + MINUTE / 2, 60), sample(3 * MINUTE + MINUTE / 2, 90)]

        expect(perSlotOf(samples, window, throttled)).toEqual([30, 45, 15, undefined])
    })

    it("reads the last minute as a rate, over the reads that span it", () => {
        const samples = [sample(0, 0), sample(MINUTE / 2, 20), sample(MINUTE, 40), sample(MINUTE + 15_000, 50)]

        expect(lastMinuteOf(samples, throttled)).toBeCloseTo(50 * MINUTE / 75_000)
        expect(lastMinuteOf([sample(0, 0)], throttled)).toBeUndefined()
        expect(sinceOf(samples, throttled)).toBe(50)
    })

    it("names the reasons a click is refused for, the known ones first", () => {
        const read = sampleOf(metricsAt(0, 1))
        read.metrics.push(new Metric({name: "frozen_clicks", value: {case: "counter", value: 1}}))
        read.metrics.push(new Metric({name: "allegiance_clicks", value: {case: "counter", value: 1}}))

        expect(reasonsOf([read])).toEqual(["throttled_clicks", "banned_clicks", "blocked_clicks", "frozen_clicks", "allegiance_clicks"])
    })

    it("reads how many boxes were caught within a bound between the first read and the last", () => {
        const samples = [sampleOf(metricsAt(-60, 59)), sampleOf(metricsAt(0, 60))]

        expect(shareUnderOf(samples, "bonus_catch_seconds", 1)).toBeCloseTo(3 / 12)
        expect(shareUnderOf(samples.slice(1), "bonus_catch_seconds", 1)).toBeUndefined()
    })
})
