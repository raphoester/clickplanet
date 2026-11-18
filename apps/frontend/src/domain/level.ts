import {PlayerLevel} from "../backends/progression.ts"
import {ChargeKind, UnlockLevels} from "./bonus.ts"

// The bar between the level's threshold and the next one's. The server sends the XP as it counts it, which can pass
// the next threshold until the raise lands, or sit under the level's own after a stricter scale: the bar clamps.
export type LevelBar = {
    // From 0 to 1.
    fill: number
    // The XP gained on this level, and what the level asks in all.
    gained: number
    span: number
    // The XP still to gain to the next level, 0 once it is passed.
    left: number
}

export function barOf({xp, levelXp, nextLevelXp}: PlayerLevel): LevelBar {
    const span = Math.max(0, nextLevelXp - levelXp)
    const gained = Math.min(span, Math.max(0, xp - levelXp))
    return {fill: span === 0 ? 1 : gained / span, gained, span, left: span - gained}
}

// What the page shows after a read: a read sent before a raise or a gain was heard never takes the level, or the XP on
// its bar, back down.
export function newerLevel(shown: PlayerLevel | undefined, heard: PlayerLevel | undefined): PlayerLevel | undefined {
    if (!heard) return shown
    if (shown && heard.level < shown.level) return shown
    if (shown && heard.level === shown.level && heard.xp < shown.xp) return shown
    return heard
}

// What the page shows after a gain: the XP moves on the level shown, and the level itself only with the levelReached
// the same facts bring right after, which says what it unlocked. Past the level shown, the bar shows full meanwhile.
export function gainedOn(shown: PlayerLevel | undefined, heard: PlayerLevel | undefined): PlayerLevel | undefined {
    if (!shown || !heard) return shown
    if (heard.level > shown.level) return heard.xp > shown.xp ? {...shown, xp: heard.xp} : shown
    return newerLevel(shown, heard)
}

export type Unlock = {kind: ChargeKind, level: number}

// The next kind the ladder unlocks above the level, none once every kind is unlocked.
export function nextUnlock(unlocks: UnlockLevels, level: number): Unlock | undefined {
    return ladderOf(unlocks).find((unlock) => unlock.level > level)
}

// The kinds a raise from one level to another unlocked, in the ladder's order: a jump may pass several.
export function unlockedBy(unlocks: UnlockLevels, from: number, to: number): Unlock[] {
    return ladderOf(unlocks).filter((unlock) => unlock.level > from && unlock.level <= to)
}

function ladderOf(unlocks: UnlockLevels): Unlock[] {
    return Object.entries(unlocks)
        .flatMap(([kind, level]) => level === undefined ? [] : [{kind: kind as ChargeKind, level}])
        .sort((a, b) => a.level - b.level)
}
