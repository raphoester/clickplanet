import {NameColor} from "../backends/player.ts"

export type PickedColor = Exclude<NameColor, NameColor.UNSPECIFIED>

export type NameColorChoice = {
    color: PickedColor
    hue: number
}

export const NAME_COLORS: readonly NameColorChoice[] = [
    {color: NameColor.RED, hue: 0},
    {color: NameColor.ORANGE, hue: 28},
    {color: NameColor.YELLOW, hue: 52},
    {color: NameColor.LIME, hue: 80},
    {color: NameColor.GREEN, hue: 125},
    {color: NameColor.TEAL, hue: 165},
    {color: NameColor.CYAN, hue: 190},
    {color: NameColor.BLUE, hue: 215},
    {color: NameColor.INDIGO, hue: 240},
    {color: NameColor.VIOLET, hue: 270},
    {color: NameColor.MAGENTA, hue: 300},
    {color: NameColor.PINK, hue: 330},
]

export function hueOf(color: NameColor | undefined): number | undefined {
    return NAME_COLORS.find((choice) => choice.color === color)?.hue
}
