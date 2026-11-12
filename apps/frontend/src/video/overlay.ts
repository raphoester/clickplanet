import {DISPLAY_FONT_DRAWS} from "../clip/names.ts"
import {VideoJob, VIDEO_PLAY_SECONDS} from "./job.ts"

// The official clips' look, drawn on the canvas: their sizes are in the pixels of their 540-wide page.
const CLIP_WIDTH = 540

// What TikTok and the others draw over: the top's tabs, the buttons down the right, the name and caption at the bottom.
const SAFE_TOP = 136
const SAFE_SIDE = 44
const SAFE_RIGHT = 108
const SAFE_BOTTOM = 210

const FADE_SECONDS = 0.5

export type Look = {
    display: string
    text: string
    ink: string
    white: string
    gold: string
    gain: string
    loss: string
    panel: string
    scrim: string
}

export type Pictures = {
    logo: HTMLImageElement
    flag: HTMLImageElement
    // The flag's own width over its height.
    flagAspect: number
}

export type Moment = {
    seconds: number
    // The share of the 24 hours played.
    played: number
    // The flag's tiles in the story's country.
    held: number
}

export function lookOfThePage(): Look {
    const style = getComputedStyle(document.documentElement)
    const token = (name: string) => style.getPropertyValue(name).trim()
    return {
        display: token("--font-display"),
        text: token("--font-text"),
        ink: token("--ink"),
        white: token("--text"),
        gold: token("--gold"),
        gain: token("--gain"),
        loss: token("--loss"),
        panel: token("--panel"),
        scrim: token("--scrim"),
    }
}

// Every display word in one face: Luckiest Guy draws Latin-1 alone, so one name in another script sets them all in Rubik.
export function displayFontOf(job: VideoJob, look: Look): (size: number) => string {
    const said = [job.headline, job.maker ?? ""]
    return said.every((text) => DISPLAY_FONT_DRAWS.test(text))
        ? (size) => `${size}px ${look.display}`
        : (size) => `900 ${size}px ${look.text}`
}

// A share of the country, to a tenth of a point, never more than it is.
function shareOf(count: number, ground: number): string {
    return `${(Math.floor(count / Math.max(1, ground) * 1000) / 10).toFixed(1)}%`
}

export function createOverlay(context: CanvasRenderingContext2D, job: VideoJob, look: Look, pictures: Pictures): (moment: Moment) => void {
    const {width, height} = context.canvas
    const u = width / CLIP_WIDTH
    const display = displayFontOf(job, look)
    const {story} = job

    const outlined = (text: string, x: number, y: number, fill: string, stroke: number, drop = 0) => {
        context.lineJoin = "round"
        context.lineWidth = stroke
        context.strokeStyle = look.ink
        if (drop > 0) {
            context.fillStyle = look.ink
            context.strokeText(text, x, y + drop)
            context.fillText(text, x, y + drop)
        }
        context.strokeText(text, x, y)
        context.fillStyle = fill
        context.fillText(text, x, y)
    }

    const framedFlag = (x: number, y: number, flagHeight: number, border: number, radius: number) => {
        const flagWidth = flagHeight * pictures.flagAspect
        context.save()
        context.beginPath()
        context.roundRect(x, y, flagWidth, flagHeight, radius)
        context.clip()
        context.drawImage(pictures.flag, x, y, flagWidth, flagHeight)
        context.restore()
        context.lineWidth = border
        context.strokeStyle = look.ink
        context.beginPath()
        context.roundRect(x, y, flagWidth, flagHeight, radius)
        context.stroke()
        return flagWidth
    }

    const fitted = (text: string, font: (size: number) => string, size: number, room: number) => {
        context.font = font(size)
        const measured = context.measureText(text).width
        return measured <= room ? size : Math.floor(size * room / measured)
    }

    // The headline, wrapped in the room the sides leave, the flag before its first word.
    const headSize = 34 * u
    const room = width - 2 * SAFE_SIDE * u
    const flagGap = 8 * u
    const headFlag = 0.75 * headSize * pictures.flagAspect + flagGap
    context.font = display(headSize)
    const lines: string[] = []
    for (const word of job.headline.split(" ")) {
        const last = lines.at(-1)
        const lead = lines.length === 1 ? headFlag : 0
        if (last !== undefined && context.measureText(`${last} ${word}`).width + lead <= room) lines[lines.length - 1] = `${last} ${word}`
        else lines.push(word)
    }
    const lineHeight = 1.05 * headSize
    const headBottom = SAFE_TOP * u + lines.length * lineHeight

    const drawTop = () => {
        const fade = context.createLinearGradient(0, 0, 0, headBottom + 32 * u)
        fade.addColorStop(0, look.scrim)
        fade.addColorStop(1, "transparent")
        context.fillStyle = fade
        context.fillRect(0, 0, width, headBottom + 32 * u)

        context.font = display(headSize)
        context.textAlign = "left"
        context.textBaseline = "alphabetic"
        lines.forEach((line, i) => {
            const lead = i === 0 ? headFlag : 0
            const left = (width - context.measureText(line).width - lead) / 2
            const baseline = SAFE_TOP * u + (i + 0.8) * lineHeight
            if (i === 0) framedFlag(left, baseline - 0.75 * headSize, 0.75 * headSize, 2 * u, 5 * u)
            outlined(line, left + lead, baseline, look.white, 5 * u, 3 * u)
        })
    }

    // The block above the bottom fifth and left of the buttons: the flag's share of the country, the address, the clock.
    const blockLeft = SAFE_SIDE * u
    const blockWidth = width - (SAFE_SIDE + SAFE_RIGHT) * u
    const blockMiddle = blockLeft + blockWidth / 2
    const trackTop = height - SAFE_BOTTOM * u - 4 * u
    const markHeight = 28 * u
    const markTop = trackTop - 8 * u - markHeight
    const counterHeight = 38 * u
    const counterTop = markTop - 8 * u - counterHeight

    const pill = (left: number, top: number, pillWidth: number, pillHeight: number) => {
        context.beginPath()
        context.roundRect(left, top, pillWidth, pillHeight, pillHeight / 2)
        context.fillStyle = look.panel
        context.fill()
        context.lineWidth = 2 * u
        context.strokeStyle = look.ink
        context.stroke()
    }

    const drawCounter = (held: number) => {
        const size = 24 * u
        const share = shareOf(held, story.tiles)
        const moved = held - story.start
        const by = shareOf(Math.abs(moved), story.tiles)
        const delta = moved === 0 || by === "0.0%" ? "" : `${moved > 0 ? "+" : "−"}${by}`
        context.font = display(size)
        const flagWidth = 20 * u * pictures.flagAspect
        const shareWidth = context.measureText(share).width
        const deltaWidth = delta ? context.measureText(delta).width + 8 * u : 0
        const pillWidth = 14 * u + flagWidth + 8 * u + shareWidth + deltaWidth + 14 * u
        const left = blockMiddle - pillWidth / 2
        pill(left, counterTop, pillWidth, counterHeight)
        framedFlag(left + 14 * u, counterTop + (counterHeight - 20 * u) / 2, 20 * u, 1.5 * u, 3 * u)
        context.textAlign = "left"
        context.textBaseline = "middle"
        context.fillStyle = look.white
        const middle = counterTop + counterHeight / 2 + 2 * u
        context.fillText(share, left + 14 * u + flagWidth + 8 * u, middle)
        if (delta) {
            context.fillStyle = moved > 0 ? look.gain : look.loss
            context.fillText(delta, left + 14 * u + flagWidth + 8 * u + shareWidth + 8 * u, middle)
        }
    }

    const markFont = (size: number) => `700 ${size}px ${look.text}`
    const markSize = fitted(job.address, markFont, 15 * u, blockWidth - 46 * u)
    const drawMark = () => {
        context.font = markFont(markSize)
        const textWidth = context.measureText(job.address).width
        const pillWidth = 6 * u + 20 * u + 6 * u + textWidth + 12 * u
        const left = blockMiddle - pillWidth / 2
        pill(left, markTop, pillWidth, markHeight)
        context.drawImage(pictures.logo, left + 6 * u, markTop + (markHeight - 20 * u) / 2, 20 * u, 20 * u)
        context.textAlign = "left"
        context.textBaseline = "middle"
        context.fillStyle = look.gold
        context.fillText(job.address, left + 32 * u, markTop + markHeight / 2 + u)
    }

    const drawTrack = (played: number) => {
        const trackWidth = blockWidth * 0.6
        const left = blockMiddle - trackWidth / 2
        context.fillStyle = look.panel
        context.beginPath()
        context.roundRect(left, trackTop, trackWidth, 4 * u, 3 * u)
        context.fill()
        context.fillStyle = look.gold
        context.beginPath()
        context.roundRect(left, trackTop, trackWidth * played, 4 * u, 3 * u)
        context.fill()
    }

    // The end card: the flag, the maker, and the address big.
    const nameRoom = width - 2 * SAFE_SIDE * u
    const makerSize = job.maker === undefined ? 0 : fitted(job.maker, display, 46 * u, nameRoom)
    const addressFont = (size: number) => `800 ${size}px ${look.text}`
    const addressSize = fitted(job.address, addressFont, 32 * u, nameRoom)
    const drawEnd = (over: number) => {
        context.globalAlpha = over
        const fade = context.createLinearGradient(0, 0, 0, height)
        fade.addColorStop(0, look.scrim)
        fade.addColorStop(0.45, look.scrim)
        fade.addColorStop(0.7, "transparent")
        context.fillStyle = fade
        context.fillRect(0, 0, width, height)

        let y = (SAFE_TOP + 24) * u
        const flagWidth = 180 * u
        const flagHeight = flagWidth / pictures.flagAspect
        framedFlag((width - flagWidth) / 2, y, flagHeight, 4 * u, 10 * u)
        y += flagHeight + 18 * u

        context.textAlign = "center"
        context.textBaseline = "alphabetic"
        if (job.maker !== undefined) {
            context.font = display(makerSize)
            y += makerSize
            outlined(job.maker, width / 2, y, look.gold, 6 * u)
            y += 14 * u
        }
        context.font = addressFont(addressSize)
        y += 18 * u + addressSize
        outlined(job.address, width / 2, y, look.white, 5 * u)
        context.globalAlpha = 1
    }

    return ({seconds, played, held}) => {
        const over = Math.min(1, Math.max(0, (seconds - VIDEO_PLAY_SECONDS) / FADE_SECONDS))
        context.globalAlpha = 1 - over
        drawTop()
        drawCounter(held)
        drawTrack(played)
        context.globalAlpha = 1
        drawMark()
        if (over > 0) drawEnd(over)
    }
}
