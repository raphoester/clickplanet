import {PlayerTitle} from "../backends/player.ts"
import {ClosedRound} from "../backends/standings.ts"
import {Battle, battleKeyOf} from "./battle.ts"
import {revealKeyOf} from "./roundReveal.ts"

export type Announcement =
    | {kind: "title", title: PlayerTitle}
    | {kind: "battle", battle: Battle}
    | {kind: "round", closed: ClosedRound}

export function announcementKeyOf(announcement: Announcement): string {
    switch (announcement.kind) {
        case "title":
            return `title:${announcement.title.id}`
        case "battle":
            return `battle:${battleKeyOf(announcement.battle)}`
        case "round":
            return `round:${revealKeyOf(announcement.closed)}`
    }
}

export function announced(queue: readonly Announcement[], next: Announcement): readonly Announcement[] {
    const key = announcementKeyOf(next)
    if (queue.some((announcement) => announcementKeyOf(announcement) === key)) return queue
    return [...queue, next]
}
