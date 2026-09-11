import {CapturedFrame} from "../viewer/capture.ts";
import {regions} from "../viewer/atlas.ts";
import {ATLAS_URL} from "../viewer/atlasAsset.ts";
import {cardSize, fitInBox, shareLabel, ShareStats, statsLine} from "../../domain/shareCard.ts";

/**
 * The globe's own pixels with the player's standing laid over them, composed on
 * a 2D canvas.
 *
 * Nothing here screenshots the page: the menu is a translucent panel over the
 * globe with a leaderboard scrolling inside it, and what makes sense to look at
 * makes a poor picture. The badge is drawn from the same numbers the menu is
 * drawn from, at a size that reads wherever the image is posted.
 */

const TITLE_FONT = '"Luckiest Guy", sans-serif'
const LABEL_FONT = 'Oswald, sans-serif'

export async function drawShareCard(frame: CapturedFrame, stats: ShareStats): Promise<Blob> {
    const size = cardSize(frame.width, frame.height)

    const canvas = document.createElement("canvas")
    canvas.width = size.width
    canvas.height = size.height

    const context = canvas.getContext("2d")
    if (!context) throw new Error("this browser would not give a 2D context for the share image")

    drawGlobe(context, frame, size)

    // Both are wanted before the first `measureText`: a fallback face measures
    // differently, and a badge laid out against one and drawn in the other has
    // the panel in the wrong place.
    const [atlas] = await Promise.all([loadAtlas(), fontsReady()])

    drawBadge(context, size, stats, atlas)
    drawLink(context, size, stats)

    return encode(canvas)
}

function drawGlobe(
    context: CanvasRenderingContext2D,
    frame: CapturedFrame,
    size: {width: number, height: number},
) {
    const source = document.createElement("canvas")
    source.width = frame.width
    source.height = frame.height
    source.getContext("2d")?.putImageData(
        new ImageData(frame.pixels, frame.width, frame.height), 0, 0)

    context.imageSmoothingEnabled = true
    context.imageSmoothingQuality = "high"
    context.drawImage(source, 0, 0, size.width, size.height)
}

function drawBadge(
    context: CanvasRenderingContext2D,
    size: {width: number, height: number},
    stats: ShareStats,
    atlas: HTMLImageElement,
) {
    // Everything is in hundredths of the shortest edge, so the badge is the same
    // size relative to the globe on a phone in portrait as on a wide desktop.
    const unit = Math.min(size.width, size.height) / 100
    const margin = 4 * unit
    const padding = 3.2 * unit
    const gap = 2.2 * unit
    let nameSize = 7 * unit
    const labelSize = 3 * unit

    const region = regions.get(stats.country.code)
    const flag = region
        ? fitInBox(region, {width: 9 * unit, height: 6 * unit})
        : {width: 0, height: 0}

    context.textAlign = "left"

    // The longest name in `countries.json` is 13 characters and fits a phone's
    // card with room to spare, so this never fires on today's data — it is what
    // stops a longer name added there from running the panel off the edge.
    // Shrunk rather than cut: a country that reads as another country is worse
    // than a country drawn small.
    const flagRun = flag.width > 0 ? flag.width + gap : 0
    const room = size.width - margin * 2 - padding * 2 - flagRun
    context.font = `${nameSize}px ${TITLE_FONT}`
    const natural = context.measureText(stats.country.name).width
    if (natural > room) {
        nameSize *= room / natural
        context.font = `${nameSize}px ${TITLE_FONT}`
    }
    const nameWidth = context.measureText(stats.country.name).width

    const label = statsLine(stats)
    context.font = `${labelSize}px ${LABEL_FONT}`
    tracking(context, 0.18 * unit)
    const labelWidth = context.measureText(label).width
    tracking(context, 0)

    const topRow = Math.max(flag.height, nameSize)
    const width = Math.max(flagRun + nameWidth, labelWidth) + padding * 2
    const height = topRow + gap + labelSize + padding * 2
    const left = margin
    const top = size.height - margin - height

    panel(context, left, top, width, height, 2.6 * unit, unit)

    const middle = top + padding + topRow / 2
    if (region && flag.width > 0) {
        context.drawImage(atlas,
            region.x, region.y, region.width, region.height,
            left + padding, middle - flag.height / 2, flag.width, flag.height)
    }

    shadow(context, unit)
    context.fillStyle = "#FFFFFF"
    context.textBaseline = "middle"
    context.font = `${nameSize}px ${TITLE_FONT}`
    context.fillText(stats.country.name, left + padding + flagRun, middle)

    clearShadow(context)
    context.fillStyle = "#FFFFFFB3"
    context.textBaseline = "top"
    context.font = `${labelSize}px ${LABEL_FONT}`
    tracking(context, 0.18 * unit)
    context.fillText(label, left + padding, top + padding + topRow + gap)
    tracking(context, 0)
}

/**
 * The link is *drawn into the image* rather than only attached to it: a picture
 * is what survives being reposted, and the whole point of this button is that
 * whoever sees it can get here.
 *
 * Opposite the badge rather than beside it, so the two never have to share a
 * width — a long country name already fills the bottom of a phone's card.
 */
function drawLink(
    context: CanvasRenderingContext2D,
    size: {width: number, height: number},
    stats: ShareStats,
) {
    const unit = Math.min(size.width, size.height) / 100

    shadow(context, unit)
    context.fillStyle = "#FFFFFF"
    context.textAlign = "right"
    context.textBaseline = "top"
    // The label face, not the display one: the page's title font has no
    // lowercase, and a query parameter drawn as "?C=PS" is a link that does not
    // work for whoever retypes it.
    context.font = `500 ${3.2 * unit}px ${LABEL_FONT}`
    tracking(context, 0.1 * unit)
    context.fillText(shareLabel(stats.country.code), size.width - 4 * unit, 4 * unit)
    tracking(context, 0)
    clearShadow(context)
}

function panel(
    context: CanvasRenderingContext2D,
    x: number, y: number, width: number, height: number,
    radius: number,
    unit: number,
) {
    context.beginPath()
    if (context.roundRect) context.roundRect(x, y, width, height, radius)
    else context.rect(x, y, width, height)

    context.fillStyle = "#000000B8"
    context.fill()
    context.lineWidth = Math.max(1, 0.2 * unit)
    context.strokeStyle = "#FFFFFF33"
    context.stroke()
}

/** The page's `text-shadow: -1px 1px 0 #000000`, scaled with the card. */
function shadow(context: CanvasRenderingContext2D, unit: number) {
    context.shadowColor = "#000000"
    context.shadowOffsetX = -0.15 * unit
    context.shadowOffsetY = 0.15 * unit
    context.shadowBlur = 0.4 * unit
}

function clearShadow(context: CanvasRenderingContext2D) {
    context.shadowColor = "transparent"
    context.shadowOffsetX = 0
    context.shadowOffsetY = 0
    context.shadowBlur = 0
}

/** `letterSpacing` is new enough that not every browser carries it, and one
 *  without it simply draws the label a shade tighter. */
function tracking(context: CanvasRenderingContext2D, pixels: number) {
    (context as {letterSpacing?: string}).letterSpacing = `${pixels}px`
}

let atlas: Promise<HTMLImageElement> | undefined

/** The same sprite sheet the globe and every flag in the menu are cut from, so
 *  the card cannot show a flag the page does not have. Same origin, so the
 *  canvas it is drawn into stays readable. */
function loadAtlas(): Promise<HTMLImageElement> {
    atlas ??= new Promise((resolve, reject) => {
        const image = new Image()
        image.onload = () => resolve(image)
        image.onerror = () => {
            atlas = undefined
            reject(new Error(`the flag atlas at ${ATLAS_URL} could not be loaded`))
        }
        image.src = ATLAS_URL
    })

    return atlas
}

/** The page is already drawing in both faces, so this is only ever a wait on a
 *  first paint that has not finished — but a card laid out in the fallback face
 *  is a card with the text hanging out of its panel. */
async function fontsReady(): Promise<void> {
    await document.fonts?.ready
}

function encode(canvas: HTMLCanvasElement): Promise<Blob> {
    return new Promise((resolve, reject) => {
        canvas.toBlob(
            (blob) => blob ? resolve(blob) : reject(new Error("the share image could not be encoded")),
            "image/png")
    })
}
