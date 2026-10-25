import type {Point} from "./tilePeek.ts"

export type Size = {width: number, height: number}

const POINTER_GAP_PX = 16
const FINGER_GAP_PX = 48
const EDGE_PX = 8

export function cardPlacement(point: Point, card: Size, view: Size, touch: boolean): Point {
    const x = touch
        ? point.x - card.width / 2
        : beside(point.x, card.width, view.width, POINTER_GAP_PX)
    const y = touch
        ? above(point.y, card.height, FINGER_GAP_PX)
        : beside(point.y, card.height, view.height, POINTER_GAP_PX)

    return {
        x: clamp(x, EDGE_PX, view.width - card.width - EDGE_PX),
        y: clamp(y, EDGE_PX, view.height - card.height - EDGE_PX),
    }
}

function beside(at: number, length: number, room: number, gap: number): number {
    const after = at + gap
    return after + length <= room - EDGE_PX ? after : at - gap - length
}

function above(at: number, length: number, gap: number): number {
    const before = at - gap - length
    return before >= EDGE_PX ? before : at + gap
}

function clamp(value: number, low: number, high: number): number {
    return Math.max(low, Math.min(value, high))
}
