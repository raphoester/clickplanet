import {NameColor} from "../backends/player.ts"

export type NameColorChoice = {
    color: NameColor
    label: string
    hue: number
}

export const NAME_COLORS: readonly NameColorChoice[] = [
    {color: NameColor.RED, label: "Red", hue: 0},
    {color: NameColor.ORANGE, label: "Orange", hue: 28},
    {color: NameColor.YELLOW, label: "Yellow", hue: 52},
    {color: NameColor.LIME, label: "Lime", hue: 80},
    {color: NameColor.GREEN, label: "Green", hue: 125},
    {color: NameColor.TEAL, label: "Teal", hue: 165},
    {color: NameColor.CYAN, label: "Cyan", hue: 190},
    {color: NameColor.BLUE, label: "Blue", hue: 215},
    {color: NameColor.INDIGO, label: "Indigo", hue: 240},
    {color: NameColor.VIOLET, label: "Violet", hue: 270},
    {color: NameColor.MAGENTA, label: "Magenta", hue: 300},
    {color: NameColor.PINK, label: "Pink", hue: 330},
]

export function hueOf(color: NameColor | undefined): number | undefined {
    return NAME_COLORS.find((choice) => choice.color === color)?.hue
}
