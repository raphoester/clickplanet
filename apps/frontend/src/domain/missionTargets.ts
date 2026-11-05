import {DayMissions, FlagMissionOrder, MissionOrder} from "../backends/missions.ts"
import {dot, Point, pointOf} from "./clip/geometry.ts"
import {FrontChanges} from "./front.ts"
import {Territory} from "./territory.ts"

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

export type OpenMission = {
    id: string
    order: MissionOrder | FlagMissionOrder
    flag: boolean
}

function openOf(day: DayMissions): OpenMission[] {
    return [
        ...day.missions.filter((mission) => !mission.done).map(({id, order}) => ({id, order, flag: false})),
        ...(day.flagMission && !day.flagMission.done ? [{id: day.flagMission.id, order: day.flagMission.order, flag: true}] : []),
    ]
}

export function openTargetsOf(day: DayMissions): MissionTarget[] {
    return openOf(day).map(({order}) => targetOf(order)).filter((target) => target !== undefined)
}

function aimedBy(day: DayMissions, aims: (target: MissionTarget) => boolean): OpenMission[] {
    return openOf(day).filter(({order}) => {
        const target = targetOf(order)
        return target !== undefined && aims(target)
    })
}

// What the map draws on it: its ring, or the outline of its ground.
export function missionsOnTile(day: DayMissions, tile: TileFacts): OpenMission[] {
    return aimedBy(day, (target) => target.ground !== undefined && tile.ground === target.ground || counts(target, tile))
}

export function missionsOnTerritory(day: DayMissions, territory: Pick<Territory, "ground" | "holdings">): OpenMission[] {
    return aimedBy(day, (target) => target.ground !== undefined
        ? territory.ground === target.ground
        : territory.holdings.some(({flag}) => flag === target.flag))
}

export function groundsOf(targets: readonly MissionTarget[]): Set<string> {
    return new Set(targets.map((target) => target.ground).filter((ground) => ground !== undefined))
}

export type MissionGround = {
    ownerOf(tile: number): string | undefined
    shieldsOf(tile: number): number
    groundOf(tile: number): string | undefined
}

export type TileFacts = {
    owner: string | undefined
    ground: string | undefined
    shields: number
}

function factsOf(tile: number, ground: MissionGround): TileFacts {
    return {owner: ground.ownerOf(tile), ground: ground.groundOf(tile), shields: ground.shieldsOf(tile)}
}

function counts(target: MissionTarget, tile: TileFacts): boolean {
    if (target.flag === undefined || tile.owner !== target.flag) return false
    if (target.ground !== undefined && tile.ground !== target.ground) return false
    return !target.shielded || tile.shields > 0
}

function playable(target: MissionTarget, tile: number, ground: MissionGround, own: string): boolean {
    if (target.flag !== undefined) return counts(target, factsOf(tile, ground))
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
        const facts = factsOf(tile, this.ground)
        return this.targets.some((target) => counts(target, facts))
    }
}
