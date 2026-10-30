import {Appearance} from "../backends/clips.ts"
import {FlagMission} from "../backends/missions.ts"
import {PlayerTitle} from "../backends/player.ts"
import {Battle, battleKeyOf, Finale} from "./battle.ts"
import {Reveal, revealKeyOf} from "./roundReveal.ts"

export type Announcement =
    | {kind: "title", title: PlayerTitle}
    | {kind: "battle", battle: Battle}
    | {kind: "finale", finale: Finale}
    | {kind: "clip", appearance: Appearance}
    | {kind: "flagMission", mission: FlagMission, country: string}
    | Reveal

export function announcementKeyOf(announcement: Announcement): string {
    switch (announcement.kind) {
        case "title":
            return `title:${announcement.title.id}`
        case "battle":
            return `battle:${battleKeyOf(announcement.battle)}`
        case "finale":
            return `finale:${announcement.finale.roundId}`
        case "clip":
            return `clip:${announcement.appearance.postId}`
        case "flagMission":
            return `flagMission:${announcement.mission.id}`
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
