export type MuteUnit = "week" | "day" | "hour" | "minute" | "second"

export type MuteLength = {unit: MuteUnit, count: number}

const UNITS: [unit: MuteUnit, seconds: number][] = [
    ["week", 7 * 24 * 3600],
    ["day", 24 * 3600],
    ["hour", 3600],
    ["minute", 60],
    ["second", 1],
]

// In the largest unit that counts it whole.
export function muteLengthOf(seconds: number): MuteLength {
    const [unit, size] = UNITS.find(([, size]) => seconds >= size && seconds % size === 0) ?? ["second", 1]
    return {unit, count: Math.round(seconds / size)}
}
