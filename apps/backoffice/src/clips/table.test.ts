import {describe, expect, it} from "vitest"
import {csvOf, FIXED_COLUMNS, tableOf} from "./table.ts"

// A row as `clips stats` writes it, as the backend's jsonb gives it back.
const tiktok = JSON.stringify({
    version: 1, takenAt: "2026-10-24T05:41:07Z", postId: "post-1", platform: "tiktok", accountId: "tt-1",
    platformPostId: "75", url: "https://www.tiktok.com/@x/video/75", publishedAt: "2026-10-22T16:40:03Z", ageHours: 36.02,
    answer: {httpStatus: 202, status: "published", syncStatus: "pending"},
    analytics: {
        platform: "tiktok", syncStatus: "synced", errorMessage: null,
        analytics: {
            views: 812, completionRate: 0.31, impressionSources: {forYou: 0.84},
            audienceCountries: {US: 0.6, other: 0.4}, retentionCurve: {"0": 1, "1": 0.9},
        },
    },
    metadata: {storyFlag: "fr", storyCountries: ["fr", "pw"], record: {clip: {variants: {length: "short"}}}},
})

const youtube = JSON.stringify({
    version: 1, takenAt: "2026-10-24T05:41:07Z", postId: "post-1", platform: "youtube", ageHours: 36.27,
    answer: {httpStatus: 202, status: "published", syncStatus: "pending"},
    analytics: {platform: "youtube", syncStatus: "pending", analytics: null},
    retention: {
        httpStatus: 200,
        body: {
            durationSeconds: 14, retentionCurve: [
                {elapsedVideoTimeRatio: 0.5, audienceWatchRatio: 1.25, relativeRetentionPerformance: 0.6},
                {elapsedVideoTimeRatio: 1, audienceWatchRatio: 0.5, relativeRetentionPerformance: 0.8},
            ],
        },
    },
    metadata: {storyFlag: "fr"},
})

function cellsOf(table: {header: string[], records: string[][]}, row: number): Record<string, string> {
    return Object.fromEntries(table.header.map((column, i) => [column, table.records[row][i]]))
}

describe("tableOf", () => {
    it("has a column for each metric and each part of the metadata", () => {
        const table = tableOf([tiktok, youtube])

        expect(table.header.slice(0, FIXED_COLUMNS.length)).toEqual(FIXED_COLUMNS)
        const first = cellsOf(table, 0)
        expect(first.takenAt).toBe("2026-10-24T05:41:07Z")
        expect(first.ageHours).toBe("36.02")
        expect(first.httpStatus).toBe("202")
        expect(first.syncStatus).toBe("synced")
        expect(first.errorMessage).toBe("")
        expect(first["metrics.views"]).toBe("812")
        expect(first["metrics.completionRate"]).toBe("0.31")
        expect(first["metrics.impressionSources.forYou"]).toBe("0.84")
        expect(JSON.parse(first["metrics.audienceCountries"])).toEqual({US: 0.6, other: 0.4})
        expect(JSON.parse(first["metrics.retentionCurve"])).toEqual({"0": 1, "1": 0.9})
        expect(first["meta.storyFlag"]).toBe("fr")
        expect(first["meta.storyCountries"]).toBe(`["fr","pw"]`)
        expect(first["meta.record.clip.variants.length"]).toBe("short")
        expect(first["youtube.retentionStatus"]).toBe("")
    })

    it("summarizes YouTube's retention beside its curve", () => {
        const second = cellsOf(tableOf([tiktok, youtube]), 1)

        expect(second.syncStatus).toBe("pending")
        expect(second["metrics.views"]).toBe("")
        expect(second["youtube.retentionStatus"]).toBe("200")
        expect(second["youtube.durationSeconds"]).toBe("14")
        expect(second["youtube.points"]).toBe("2")
        expect(second["youtube.meanRelativeRetentionPerformance"]).toBe("0.7000")
        expect(second["youtube.retentionCurve"]).toContain(`"audienceWatchRatio":1.25`)
    })

    it("says when a curve carried was read, and why the one due is missing", () => {
        const carried = JSON.stringify({
            ...JSON.parse(youtube),
            retention: {...JSON.parse(youtube).retention, readAt: "2026-10-23T08:23:00Z", ageHours: 49.5},
            retentionMissing: "youtube's daily analytics quota is spent",
        })

        const row = cellsOf(tableOf([carried]), 0)

        expect(row["youtube.retentionReadAt"]).toBe("2026-10-23T08:23:00Z")
        expect(row["youtube.retentionAgeHours"]).toBe("49.5")
        expect(row["youtube.retentionMissing"]).toBe("youtube's daily analytics quota is spent")
        expect(row["youtube.points"]).toBe("2")
        expect(row.httpStatus).toBe("202")
    })

    it("is its header alone with no row", () => {
        expect(tableOf([])).toEqual({header: FIXED_COLUMNS, records: []})
    })
})

describe("csvOf", () => {
    it("quotes a cell with a comma, a quote or a line break", () => {
        const csv = csvOf({header: ["a", "b"], records: [["1,2", `say "hi"`], ["plain", "two\nlines"]]})

        expect(csv).toBe(`a,b\r\n"1,2","say ""hi"""\r\nplain,"two\nlines"\r\n`)
    })
})
