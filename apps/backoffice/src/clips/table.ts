// The table of the clips' numbers, made in the page from the rows the backend keeps (clips.v1.StatsRow.document, the
// row as `clips stats` wrote it): the backend sends what Zernio answered, never a table, so a column needs no change
// there. The CSV is for a spreadsheet; the rows themselves stay the source of truth.

// The columns every table starts with, in this order. The others are what the rows hold, by name.
export const FIXED_COLUMNS = [
    "takenAt", "postId", "platform", "accountId", "platformPostId", "url", "publishedAt", "ageHours",
    "httpStatus", "postStatus", "syncStatus", "errorMessage",
]

// Kept as one cell of JSON: a key a country or a second, so too many to be columns.
const WIDE_KEYS = ["retentionCurve", "audienceCountries"]

type Cells = Map<string, string>

// A column for each metric (metrics.views), each part of the metadata (meta.storyFlag,
// meta.record.clip.variants.<experiment>: an arm is a column with no code) and YouTube's retention (youtube.*). A
// list is one cell of JSON. The columns are those of all the rows, so a metric Zernio adds is a column at once.
export function tableOf(documents: string[]): {header: string[], records: string[][]} {
    const cells = documents.map(cellsOf)
    const named = new Set<string>()
    for (const row of cells) {
        for (const column of row.keys()) {
            if (!FIXED_COLUMNS.includes(column)) named.add(column)
        }
    }
    const header = [...FIXED_COLUMNS, ...[...named].sort()]
    return {header, records: cells.map(row => header.map(column => row.get(column) ?? ""))}
}

// RFC 4180: a cell with a comma, a quote or a line break is quoted, its quotes doubled.
export function csvOf({header, records}: {header: string[], records: string[][]}): string {
    return [header, ...records].map(line => line.map(quoted).join(",")).join("\r\n") + "\r\n"
}

function quoted(cell: string): string {
    return /[",\r\n]/.test(cell) ? `"${cell.replaceAll("\"", "\"\"")}"` : cell
}

type Document = {
    takenAt?: string, postId?: string, platform?: string, accountId?: string, platformPostId?: string, url?: string,
    publishedAt?: string, ageHours?: number,
    answer?: {httpStatus?: number, status?: string, syncStatus?: string},
    analytics?: {syncStatus?: string, errorMessage?: string | null, analytics?: unknown} | null,
    retention?: Retention,
    retentionMissing?: string,
    metadata?: unknown,
}

// The curve the row holds, read at its snapshot or carried from an earlier one: readAt and ageHours say when it was
// read and how old the video was then. A row written before the job kept those has neither, and was read with its row.
type Retention = {httpStatus?: number, body?: unknown, readAt?: string, ageHours?: number}

function cellsOf(text: string): Cells {
    const row = JSON.parse(text) as Document
    const cells: Cells = new Map([
        ["takenAt", row.takenAt ?? ""], ["postId", row.postId ?? ""], ["platform", row.platform ?? ""],
        ["accountId", row.accountId ?? ""], ["platformPostId", row.platformPostId ?? ""], ["url", row.url ?? ""],
        ["publishedAt", row.publishedAt ?? ""], ["ageHours", textOf(row.ageHours)],
        ["httpStatus", textOf(row.answer?.httpStatus)], ["postStatus", row.answer?.status ?? ""],
        ["syncStatus", row.answer?.syncStatus ?? ""],
    ])

    // The platform's own state wins over the post's.
    const entry = row.analytics
    if (entry !== null && entry !== undefined) {
        if (entry.syncStatus) cells.set("syncStatus", entry.syncStatus)
        if (entry.errorMessage) cells.set("errorMessage", entry.errorMessage)
        flatten(cells, "metrics", entry.analytics)
    }
    flatten(cells, "meta", row.metadata)
    if (row.retention !== undefined) retentionCells(cells, row.retention)
    // Why the curve due at this snapshot was not read: the rest of the row is whole.
    if (row.retentionMissing) cells.set("youtube.retentionMissing", row.retentionMissing)
    return cells
}

type Point = {relativeRetentionPerformance?: number}

function retentionCells(cells: Cells, retention: Retention) {
    cells.set("youtube.retentionStatus", textOf(retention.httpStatus))
    if (retention.readAt) cells.set("youtube.retentionReadAt", retention.readAt)
    if (retention.ageHours !== undefined) cells.set("youtube.retentionAgeHours", textOf(retention.ageHours))
    const body = retention.body
    if (body === null || typeof body !== "object" || Array.isArray(body)) return

    const said = body as {durationSeconds?: number, retentionCurve?: Point[]}
    cells.set("youtube.durationSeconds", textOf(said.durationSeconds))
    const curve = said.retentionCurve ?? []
    if (curve.length === 0) return
    cells.set("youtube.retentionCurve", JSON.stringify(curve))
    cells.set("youtube.points", String(curve.length))

    // How it kept its viewers against videos of its length, over the whole video: 0 the worst, 0.5 the median, 1 the best.
    const measured = curve.map(point => point.relativeRetentionPerformance).filter(value => typeof value === "number")
    if (measured.length > 0) {
        const mean = measured.reduce((total, value) => total + value, 0) / measured.length
        cells.set("youtube.meanRelativeRetentionPerformance", mean.toFixed(4))
    }
}

function flatten(cells: Cells, prefix: string, value: unknown) {
    if (value === null || value === undefined) return
    if (Array.isArray(value)) {
        cells.set(prefix, JSON.stringify(value))
        return
    }
    if (typeof value === "object") {
        for (const [key, inner] of Object.entries(value)) {
            const column = `${prefix}.${key}`
            if (WIDE_KEYS.includes(key)) cells.set(column, JSON.stringify(inner))
            else flatten(cells, column, inner)
        }
        return
    }
    cells.set(prefix, textOf(value))
}

function textOf(value: unknown): string {
    if (value === undefined || value === null) return ""
    return String(value)
}
