import {PointerEvent} from "react"
import {ArrivalDay, CONTROL, Post} from "./readout.ts"

// A post under 100 views is drawn at 100: the log axis starts there.
export const LOG_FLOOR = 100

export function logTicksOf(highest: number): number[] {
    const ticks = [LOG_FLOOR, LOG_FLOOR * 10]
    while (ticks[ticks.length - 1] < highest) ticks.push(ticks[ticks.length - 1] * 10)
    return ticks
}

export function logShareOf(value: number, ticks: number[]): number {
    const low = Math.log10(ticks[0])
    const high = Math.log10(ticks[ticks.length - 1])
    return (Math.log10(Math.max(value, ticks[0])) - low) / (high - low)
}

export function experimentsOf(posts: Post[]): string[] {
    return [...new Set(posts.map(post => post.experiment))].filter(name => name !== CONTROL).sort()
}

const SLOTS = 6

// Past the sixth, an experiment is drawn in ink: a seventh hue would pass for one of the six.
export function colorOf(experiment: string, experiments: string[]): string {
    if (experiment === CONTROL) return "var(--lab-control)"
    const slot = experiments.indexOf(experiment)
    return slot >= 0 && slot < SLOTS ? `var(--lab-${slot + 1})` : "var(--text)"
}

// Lane 0 is the row's middle; the others are filled in their order.
export function lanesOf(xs: number[], gap: number, lanes = 3): number[] {
    const ends = Array<number>(lanes).fill(-Infinity)
    const lane = Array<number>(xs.length).fill(0)
    const order = xs.map((_, i) => i).sort((a, b) => xs[a] - xs[b])
    for (const i of order) {
        const free = ends.findIndex(end => xs[i] - end >= gap)
        const pick = free >= 0 ? free : ends.indexOf(Math.min(...ends))
        lane[i] = pick
        ends[pick] = xs[i]
    }
    return lane
}

export type Spot = {key: string, x: number, y: number}

export function nearestOf(spots: Spot[], x: number, y: number, reach: number): string | undefined {
    let best: Spot | undefined
    let bestDistance = reach * reach
    for (const spot of spots) {
        const distance = (spot.x - x) ** 2 + (spot.y - y) ** 2
        if (distance <= bestDistance) {
            best = spot
            bestDistance = distance
        }
    }
    return best?.key
}

export type Dot = {key: string, post: Post, x: number, y: number, color: string, young: boolean}

const REACH = 16

export function pointedOf(event: PointerEvent<Element>, dots: Dot[]): string | undefined {
    const box = event.currentTarget.getBoundingClientRect()
    const spots = dots.map(dot => ({key: dot.post.postId, x: dot.x, y: dot.y}))
    return nearestOf(spots, event.clientX - box.left, event.clientY - box.top, REACH)
}

export function everyOf(band: number, room: number): number {
    return Math.max(1, Math.ceil(room / band))
}

export function labelled(index: number, length: number, every: number): boolean {
    return (length - 1 - index) % every === 0
}

export type ArrivalBar = ArrivalDay & {filling: boolean}

export function arrivalBarsOf(days: ArrivalDay[], today: string): ArrivalBar[] {
    return [...days].sort((a, b) => a.day.localeCompare(b.day)).map(day => ({...day, filling: day.day === today}))
}

export type BarGeometry = {x: number, y: number, length: number, thickness: number}

export function rightBarOf({x, y, length, thickness}: BarGeometry): string {
    const r = Math.min(4, length, thickness / 2)
    const end = x + length
    return `M${x},${y} H${end - r} A${r},${r} 0 0 1 ${end},${y + r} V${y + thickness - r} ` +
        `A${r},${r} 0 0 1 ${end - r},${y + thickness} H${x} Z`
}

export function upBarOf({x, y, length, thickness}: BarGeometry): string {
    const r = Math.min(4, length, thickness / 2)
    const top = y - length
    return `M${x},${y} V${top + r} A${r},${r} 0 0 1 ${x + r},${top} H${x + thickness - r} ` +
        `A${r},${r} 0 0 1 ${x + thickness},${top + r} V${y} Z`
}
