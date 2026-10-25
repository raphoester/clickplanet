import type {TileHolder} from "../backends/backend.ts"

export type Point = {x: number, y: number}

export type Peek = {
    tile: number
    countryCode: string
    point: Point
    touch: boolean
}

export type Peeked = {
    tile: number
    point: Point
    touch: boolean
}

export type PeekRules = {
    restSeconds: number
    holdSeconds: number
    tolerancePx: number
}

export type PeekPointer = {
    pointerId: number
    clientX: number
    clientY: number
    isPrimary: boolean
}

type Rest = {tile: number, since: number, told: boolean, point: Point}

type Press = {pointerId: number, x: number, y: number, tile: number, since: number}

export class TilePeek {
    private rest: Rest | undefined
    private press: Press | undefined
    private peekedByPress = false

    constructor(private readonly rules: PeekRules) {}

    get peeked(): boolean {
        return this.peekedByPress
    }

    hover(tile: number | undefined, at: number, point: Point) {
        if (this.rest && tile === this.rest.tile) {
            this.rest.point = point
            return
        }
        this.rest = tile === undefined ? undefined : {tile, since: at, told: false, point}
    }

    clicked() {
        if (this.rest) this.rest.told = true
    }

    begin(pointer: PeekPointer, tile: number | undefined, at: number) {
        this.peekedByPress = false
        this.press = pointer.isPrimary && tile !== undefined
            ? {pointerId: pointer.pointerId, x: pointer.clientX, y: pointer.clientY, tile, since: at}
            : undefined
    }

    move(pointer: Pick<PeekPointer, "pointerId" | "clientX" | "clientY">) {
        if (!this.press || pointer.pointerId !== this.press.pointerId) return
        if (Math.hypot(pointer.clientX - this.press.x, pointer.clientY - this.press.y) > this.rules.tolerancePx) this.press = undefined
    }

    end(pointerId: number) {
        if (this.press?.pointerId === pointerId) this.press = undefined
    }

    tick(at: number): Peeked | undefined {
        if (this.press && at - this.press.since >= this.rules.holdSeconds - 1e-9) {
            const {tile, x, y} = this.press
            this.press = undefined
            this.peekedByPress = true
            return {tile, point: {x, y}, touch: true}
        }
        if (this.rest && !this.rest.told && at - this.rest.since >= this.rules.restSeconds - 1e-9) {
            this.rest.told = true
            return {tile: this.rest.tile, point: this.rest.point, touch: false}
        }
        return undefined
    }
}

export function holderShown(holder: TileHolder | undefined, peek: Peek): TileHolder | undefined {
    if (!holder || holder.countryCode !== peek.countryCode) return undefined
    return holder
}

const MINUTE_MS = 60_000
const HOUR_MS = 60 * MINUTE_MS
const DAY_MS = 24 * HOUR_MS

export function agoOf(at: number, now: number): string {
    const since = Math.max(0, now - at)
    if (since < MINUTE_MS) return "just now"
    if (since < HOUR_MS) return `${Math.floor(since / MINUTE_MS)} min ago`
    if (since < DAY_MS) return plural(Math.floor(since / HOUR_MS), "hour")
    return plural(Math.floor(since / DAY_MS), "day")
}

function plural(count: number, unit: string): string {
    return `${count} ${unit}${count === 1 ? "" : "s"} ago`
}
