import {Timestamp} from "@bufbuild/protobuf"
import {GetMetricsResponse, Metric} from "../gen/grpc/metrics/v1/admin_pb.ts"
import {Window} from "../home/ledger.ts"

export const MINUTE = 60_000

// One answer of GetMetrics: when the server read it, and from when its counters count.
export type Sample = {at: number, startedAt: number, metrics: Metric[]}

export type Select = (metric: Metric) => boolean

function millisOf(at: Timestamp | undefined): number {
    return at === undefined ? 0 : Number(at.seconds) * 1000 + at.nanos / 1e6
}

export function sampleOf(answer: GetMetricsResponse): Sample {
    return {at: millisOf(answer.readAt), startedAt: millisOf(answer.startedAt), metrics: answer.metrics}
}

// The samples of the span, the new one last. One before the span stays, so its first minute has a start.
export function keptSamples(samples: Sample[], next: Sample, spanMs: number): Sample[] {
    const last = samples[samples.length - 1]
    const all = last === undefined || next.at > last.at ? [...samples, next] : samples
    const cut = all.findIndex(sample => sample.at >= next.at - spanMs)
    return all.slice(Math.max(0, cut - 1))
}

export function named(name: string, labels: Record<string, string> = {}): Select {
    return metric => metric.name === name
        && Object.entries(labels).every(([key, value]) => metric.labels.some(label => label.name === key && label.value === value))
}

// The sum of the counters and gauges picked.
export function valueOf(sample: Sample, select: Select): number {
    let sum = 0
    for (const metric of sample.metrics) {
        if (!select(metric)) continue
        if (metric.value.case === "counter" || metric.value.case === "gauge") sum += metric.value.value
    }
    return sum
}

// What the counters picked counted from one sample to the next. A restart starts them again: all they hold is new.
export function increaseOf(before: Sample, after: Sample, select: Select): number {
    const now = valueOf(after, select)
    return after.startedAt === before.startedAt ? Math.max(0, now - valueOf(before, select)) : now
}

// What the counters picked counted in each slot of the window, spread evenly over the time between two samples.
// A slot no two samples span is unknown.
export function perSlotOf(samples: Sample[], {since, count, step}: Window, select: Select): (number | undefined)[] {
    const values: (number | undefined)[] = Array.from({length: count}, () => undefined)
    const start = since.getTime()

    for (let i = 1; i < samples.length; i++) {
        const from = samples[i - 1].at
        const to = samples[i].at
        const increase = increaseOf(samples[i - 1], samples[i], select)
        const first = Math.max(0, Math.floor((from - start) / step))
        const last = Math.min(count - 1, Math.floor((to - start) / step))

        for (let slot = first; slot <= last; slot++) {
            const slotStart = start + slot * step
            const overlap = Math.min(to, slotStart + step) - Math.max(from, slotStart)
            if (overlap <= 0) continue
            values[slot] = (values[slot] ?? 0) + increase * overlap / (to - from)
        }
    }

    return values
}

// What the counters picked counted over the last minute of samples, as a rate a minute. Unknown with one sample.
export function lastMinuteOf(samples: Sample[], select: Select): number | undefined {
    const latest = samples[samples.length - 1]
    if (latest === undefined) return undefined

    let from = samples.length - 1
    while (from > 0 && latest.at - samples[from].at < MINUTE) from--
    if (from === samples.length - 1 || latest.at <= samples[from].at) return undefined

    return sinceOf(samples.slice(from), select) * MINUTE / (latest.at - samples[from].at)
}

// What the counters picked counted from the first sample to the last.
export function sinceOf(samples: Sample[], select: Select): number {
    let sum = 0
    for (let i = 1; i < samples.length; i++) sum += increaseOf(samples[i - 1], samples[i], select)
    return sum
}

// The gauges picked, as the latest sample reads them.
export function nowOf(samples: Sample[], select: Select): number | undefined {
    const latest = samples[samples.length - 1]
    return latest === undefined ? undefined : valueOf(latest, select)
}

// The share of the observations under `bound` from the first sample to the last, of the histogram named.
export function shareUnderOf(samples: Sample[], name: string, bound: number): number | undefined {
    let under = 0
    let all = 0
    for (let i = 1; i < samples.length; i++) {
        const before = histogramOf(samples[i - 1], name)
        const after = histogramOf(samples[i], name)
        if (after === undefined) continue
        const restarted = before === undefined || samples[i].startedAt !== samples[i - 1].startedAt
        under += countUnder(after, bound) - (restarted ? 0 : countUnder(before, bound))
        all += Number(after.count) - (restarted ? 0 : Number(before.count))
    }
    return all <= 0 ? undefined : under / all
}

type Histogram = Extract<Metric["value"], {case: "histogram"}>["value"]

function histogramOf(sample: Sample, name: string): Histogram | undefined {
    const metric = sample.metrics.find(metric => metric.name === name)
    return metric?.value.case === "histogram" ? metric.value.value : undefined
}

function countUnder(histogram: Histogram, bound: number): number {
    const bucket = histogram.buckets.find(bucket => bucket.upperBound >= bound)
    return bucket === undefined ? Number(histogram.count) : Number(bucket.count)
}

// The reasons a click was refused for: every counter named `<reason>_clicks`, the known ones first in their order.
const KNOWN_REASONS = ["throttled_clicks", "refused_clicks", "blocked_clicks", "frozen_clicks"]
// The callers planet refuses are the ones the antibot banned.
const LABELS: Record<string, string> = {refused_clicks: "banned"}

export function reasonsOf(samples: Sample[]): string[] {
    const names = new Set<string>()
    for (const metric of samples[samples.length - 1]?.metrics ?? []) {
        if (metric.value.case === "counter" && metric.name.endsWith("_clicks")) names.add(metric.name)
    }
    const known = KNOWN_REASONS.filter(name => names.has(name))
    const others = [...names].filter(name => !KNOWN_REASONS.includes(name)).sort()
    return [...known, ...others]
}

export function reasonLabelOf(name: string): string {
    return LABELS[name] ?? name.replace(/_clicks$/, "")
}

export function refused(metric: Metric): boolean {
    return metric.value.case === "counter" && metric.name.endsWith("_clicks")
}

// Every value of one label among the series named, as the latest sample holds them.
export function labelValuesOf(samples: Sample[], name: string, label: string): string[] {
    const values = new Set<string>()
    for (const metric of samples[samples.length - 1]?.metrics ?? []) {
        if (metric.name !== name) continue
        const value = metric.labels.find(candidate => candidate.name === label)?.value
        if (value !== undefined) values.add(value)
    }
    return [...values].sort()
}

export type Kind = "counter" | "gauge"

// Every counter and gauge of the latest sample by name, its labels summed.
export function namesOf(samples: Sample[]): {name: string, kind: Kind}[] {
    const kinds = new Map<string, Kind>()
    for (const metric of samples[samples.length - 1]?.metrics ?? []) {
        if (metric.value.case === "counter" || metric.value.case === "gauge") kinds.set(metric.name, metric.value.case)
    }
    return [...kinds].map(([name, kind]) => ({name, kind})).sort((a, b) => a.name.localeCompare(b.name))
}
