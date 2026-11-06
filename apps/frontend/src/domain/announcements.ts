import {Appearance} from "../backends/clips.ts"
import {FlagMission} from "../backends/missions.ts"
import {PlayerTitle} from "../backends/progression.ts"
import {TeaserCard} from "../backends/promo.ts"
import {BattleToCome} from "../backends/turn.ts"
import {Battle, battleKeyOf} from "./battle.ts"
import {teaserKeyOf} from "./battleTeaser.ts"
import {Reveal, revealKeyOf} from "./roundReveal.ts"

export type Announcement =
    | {kind: "title", title: PlayerTitle}
    | {kind: "battle", battle: Battle}
    // A battle's teaser, due by its time rules, or asked for by the player (`asked`): then it plays no entrance.
    | {kind: "teaser", battle: BattleToCome, cards: readonly TeaserCard[], asked?: boolean}
    | {kind: "clip", appearance: Appearance}
    | {kind: "flagMission", mission: FlagMission, country: string}
    | Reveal

export function announcementKeyOf(announcement: Announcement): string {
    switch (announcement.kind) {
        case "title":
            return `title:${announcement.title.id}`
        case "battle":
            return `battle:${battleKeyOf(announcement.battle)}`
        case "teaser":
            return `teaser:${teaserKeyOf(announcement.battle)}`
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
