import type {Key} from "../i18n/translator.ts"

export const TAP_HINT_MS = 7000
export const GUIDE_MS = 10_000

export type Figure = "tiles" | "share" | "gained" | "gainedBattle" | "points" | "today" | "battle"

export const FIGURES: Record<Figure, Key> = {
    tiles: "board.figure.tiles",
    share: "board.figure.share",
    gained: "board.figure.gained",
    gainedBattle: "board.figure.gainedBattle",
    points: "board.figure.points",
    today: "board.figure.today",
    battle: "board.figure.battle",
}
