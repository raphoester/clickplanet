import {PlayerLevel} from "../backends/progression.ts"

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

// What the page shows after a read: a read sent before a raise was heard never takes the level back down.
export function newerLevel(shown: PlayerLevel | undefined, heard: PlayerLevel | undefined): PlayerLevel | undefined {
    if (!heard) return shown
    if (shown && heard.level < shown.level) return shown
    return heard
}
