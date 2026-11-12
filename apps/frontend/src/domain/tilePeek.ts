import type {TileHolder} from "../backends/backend.ts"

export type Point = {x: number, y: number}

export type Peek = {
    tile: number
    countryCode: string
    ground: string | undefined
    shields: number
    point: Point
    touch: boolean
}

export type Peeked = {
    spot: number
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

type Rest = {spot: number, since: number, told: boolean, point: Point}

type Press = {pointerId: number, x: number, y: number, spot: number, since: number}

// A spot is what the pointer asks about: a tile, or in orbit a landmass.
export class TilePeek {
    private rest: Rest | undefined
    private press: Press | undefined
    private peekedByPress = false

    constructor(private readonly rules: PeekRules) {}

    get peeked(): boolean {
        return this.peekedByPress
    }

    hover(spot: number | undefined, at: number, point: Point) {
        if (this.rest && spot === this.rest.spot) {
            this.rest.point = point
            return
        }
        this.rest = spot === undefined ? undefined : {spot, since: at, told: false, point}
    }

    clicked() {
        if (this.rest) this.rest.told = true
    }

    begin(pointer: PeekPointer, spot: number | undefined, at: number) {
        this.peekedByPress = false
        this.press = pointer.isPrimary && spot !== undefined
            ? {pointerId: pointer.pointerId, x: pointer.clientX, y: pointer.clientY, spot, since: at}
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
            const {spot, x, y} = this.press
            this.press = undefined
            this.peekedByPress = true
            return {spot, point: {x, y}, touch: true}
        }
        if (this.rest && !this.rest.told && at - this.rest.since >= this.rules.restSeconds - 1e-9) {
            this.rest.told = true
            return {spot: this.rest.spot, point: this.rest.point, touch: false}
        }
        return undefined
    }
}

export function holderShown(holder: TileHolder | undefined, peek: Peek): TileHolder | undefined {
    if (!holder || holder.countryCode !== peek.countryCode) return undefined
    return holder
}
