import {BattleToCome} from "../../backends/turn.ts"
import {Battle, endsToday} from "../../domain/battle.ts"
import {battleDayOf, teaserClockOf} from "../../domain/battleTeaser.ts"
import {t, translator} from "../../i18n/page.ts"
import {dayAndTimeRange, timeOfDay, timeRange} from "../season/dates.ts"

// What the players call a battle: the season's finale is its Final Battle.
export function battleName(battle: {endsSeason?: boolean}): string {
    return t(battle.endsSeason ? "finale.name" : "battle.name")
}

export function battleEndsLine(battle: Battle, now: number): string {
    const time = timeOfDay(battle.endsAt)
    return endsToday(battle, now)
        ? t("battle.endsAt", {time})
        : t("battle.endsOnDayAt", {day: translator().dateTime(battle.endsAt, {weekday: "short"}), time})
}

// When it plays, in the player's time, and what it is worth when it is more than a day's points.
export function battleWhenLine(battle: BattleToCome, now: number): string {
    const when = whenOf(battle, now)
    return battle.times > 1 ? t("battle.whenWorth", {when, times: battle.times}) : when
}

function whenOf(battle: BattleToCome, now: number): string {
    const day = battleDayOf(battle, now)
    switch (day) {
        case "today":
            return t("battle.today", {hours: timeRange(battle.startsAt, battle.endsAt)})
        case "tomorrow":
            return t("battle.tomorrow", {hours: timeRange(battle.startsAt, battle.endsAt)})
        case "later":
            return dayAndTimeRange(battle.startsAt, battle.endsAt)
    }
}

export function teaserClockLine(battle: BattleToCome, now: number): string {
    const clock = teaserClockOf(battle, now)
    const time = translator().countdown(clock.countdown)
    switch (clock.to) {
        case "start":
            return t("battle.startsIn", {time})
        case "end":
            return t("battle.playingEndsIn", {time})
    }
}
