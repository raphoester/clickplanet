import {PlayerTitle} from "../backends/player.ts"
import {Battle, battleKeyOf} from "./battle.ts"
import {Reveal, revealKeyOf} from "./roundReveal.ts"

export type Announcement =
    | {kind: "title", title: PlayerTitle}
    | {kind: "battle", battle: Battle}
    | Reveal

export function announcementKeyOf(announcement: Announcement): string {
    switch (announcement.kind) {
        case "title":
            return `title:${announcement.title.id}`
        case "battle":
            return `battle:${battleKeyOf(announcement.battle)}`
        case "round":
        case "over":
        case "points":
        case "players":
        case "results":
            return `${announcement.kind}:${revealKeyOf(announcement)}`
    }
}

export function announced(queue: readonly Announcement[], next: Announcement): readonly Announcement[] {
    const key = announcementKeyOf(next)
    if (queue.some((announcement) => announcementKeyOf(announcement) === key)) return queue
    return [...queue, next]
}
