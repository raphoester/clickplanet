import {DayMissions, FlagMissionOrder, MissionOrder} from "../backends/missions.ts"
import {dot, Point, pointOf} from "./clip/geometry.ts"
import {FrontChanges} from "./front.ts"

export type MissionTarget = {
    ground?: string
    flag?: string
    shielded: boolean
}

export function targetOf(order: MissionOrder | FlagMissionOrder): MissionTarget | undefined {
    switch (order.kind) {
        case "takeFrom":
            return {flag: order.country, shielded: false}
        case "takeIn":
        case "bombIn":
            return {ground: order.country, shielded: false}
        case "takeFromIn":
            return {flag: order.country, ground: order.ground, shielded: false}
        case "breakShields":
            return {flag: order.country, shielded: true}
        case "takeFor":
        case "shieldTiles":
        case "catchBoxes":
        case "answerQuizzes":
        case "fortifyTiles":
            return undefined
    }
}

export function openTargetsOf(day: DayMissions): MissionTarget[] {
    const orders = [
        ...day.missions.filter((mission) => !mission.done).map((mission) => mission.order),
        ...(day.flagMission && !day.flagMission.done ? [day.flagMission.order] : []),
    ]
    return orders.map(targetOf).filter((target) => target !== undefined)
}

export function groundsOf(targets: readonly MissionTarget[]): Set<string> {
    return new Set(targets.map((target) => target.ground).filter((ground) => ground !== undefined))
}

export type MissionGround = {
    ownerOf(tile: number): string | undefined
    shieldsOf(tile: number): number
    groundOf(tile: number): string | undefined
}

function counts(target: MissionTarget, tile: number, ground: MissionGround): boolean {
    if (target.flag === undefined || ground.ownerOf(tile) !== target.flag) return false
    if (target.ground !== undefined && ground.groundOf(tile) !== target.ground) return false
    return !target.shielded || ground.shieldsOf(tile) > 0
}

function playable(target: MissionTarget, tile: number, ground: MissionGround, own: string): boolean {
    if (target.flag !== undefined) return counts(target, tile, ground)
    return ground.groundOf(tile) === target.ground && ground.ownerOf(tile) !== own
}

export function nearestPlayable(
    target: MissionTarget, own: string, ground: MissionGround, positions: ArrayLike<number>, from: Point,
): number | undefined {
    let nearest: number | undefined
    let closest = -Infinity
    for (let tile = 1; tile * 3 <= positions.length; tile++) {
        if (!playable(target, tile, ground, own)) continue
        const along = dot(pointOf(positions, tile), from)
        if (along > closest) {
            closest = along
            nearest = tile
        }
    }
    return nearest
}

export class MissionMarks {
    private targets: readonly MissionTarget[] = []
    private readonly marked = new Set<number>()

    constructor(private readonly ground: MissionGround, private readonly size: number) {}

    follow(targets: readonly MissionTarget[]): FrontChanges {
        this.targets = targets.filter((target) => target.flag !== undefined)
        const off = [...this.marked]
        this.marked.clear()
        if (this.targets.length > 0) {
            for (let tile = 1; tile <= this.size; tile++) {
                if (this.counts(tile)) this.marked.add(tile)
            }
        }
        const on = [...this.marked]
        return {on, off: off.filter((tile) => !this.marked.has(tile))}
    }

    touch(tiles: readonly number[]): FrontChanges {
        const changes: FrontChanges = {on: [], off: []}
        if (this.targets.length === 0 && this.marked.size === 0) return changes
        for (const tile of tiles) {
            const marked = this.counts(tile)
            if (marked === this.marked.has(tile)) continue
            if (marked) {
                this.marked.add(tile)
                changes.on.push(tile)
            } else {
                this.marked.delete(tile)
                changes.off.push(tile)
            }
        }
        return changes
    }

    private counts(tile: number): boolean {
        return this.targets.some((target) => counts(target, tile, this.ground))
    }
}
