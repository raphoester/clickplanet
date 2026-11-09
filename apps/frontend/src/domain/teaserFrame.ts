import {battleNameOf} from "./battle.ts"

// What the frame of a teaser posted on TikTok and YouTube says of its battle, around the card. The post stays up long
// after the battle, so the day is written out, never "tonight", and the time is UTC, the one the schedule keeps: a post
// has no viewer's time zone.
export interface TeaserFrame {
    // "Battle" or "Final Battle", as the game names it.
    name: string
    // "Saturday 7 November".
    day: string
    // "20:00 UTC".
    time: string
}

export function teaserFrameOf(startsAt: Date, endsSeason: boolean): TeaserFrame {
    const day = new Intl.DateTimeFormat("en-GB", {weekday: "long", day: "numeric", month: "long", timeZone: "UTC"})
        .format(startsAt).replace(",", "")
    const time = new Intl.DateTimeFormat("en-GB", {hour: "2-digit", minute: "2-digit", hourCycle: "h23", timeZone: "UTC"})
        .format(startsAt)
    return {name: battleNameOf({endsSeason}), day, time: `${time} UTC`}
}
