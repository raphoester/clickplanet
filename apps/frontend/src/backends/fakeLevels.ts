import {PlayerLevel} from "./progression.ts"

// The server's curve, so the fake's bars run between the thresholds the server would send.
function thresholdOf(level: number): number {
    return 14 * (level - 1) ** 3
}

export function fakeLevelOf(xp: number): PlayerLevel {
    let level = 1
    while (thresholdOf(level + 1) <= xp) level++
    return {level, xp, levelXp: thresholdOf(level), nextLevelXp: thresholdOf(level + 1)}
}

// A made-up career's XP: between level 2 and 30, somewhere along its bar.
export function fakeXpOf(seed: number): number {
    const level = 2 + seed % 29
    const along = (seed >>> 5) % 100 / 100
    return Math.floor(thresholdOf(level) + along * (thresholdOf(level + 1) - thresholdOf(level)))
}
