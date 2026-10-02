import {CapturedFrame} from "../viewer/capture.ts";
import {regions} from "../viewer/atlas.ts";
import {ATLAS_URL} from "../viewer/atlasAsset.ts";
import {cardLayout, Crop, fitInBox, shareLabel, ShareStats, statsLine} from "../../domain/shareCard.ts";
import {TITLE_CAP_HEIGHT, TITLE_FONT_FAMILY} from "../titleFont.ts";

const TITLE_FONT = TITLE_FONT_FAMILY
const LABEL_FONT = 'Oswald, sans-serif'

const LOGO_URL = "/static/logo.svg"

export async function drawShareCard(frame: CapturedFrame, stats: ShareStats): Promise<Blob> {
    const {crop, ...size} = cardLayout(frame.width, frame.height)

    const canvas = document.createElement("canvas")
    canvas.width = size.width
    canvas.height = size.height

    const context = canvas.getContext("2d")
    if (!context) throw new Error("this browser would not give a 2D context for the share image")

    drawGlobe(context, frame, crop, size)

    // Fonts must be ready before the first measureText: a fallback face measures differently.
    const [atlas, logo] = await Promise.all([
        loadImage(ATLAS_URL),
        loadImage(LOGO_URL),
        fontsReady(),
    ])

    drawMasthead(context, size, stats, logo)
    drawBadge(context, size, stats, atlas)

    return encode(canvas)
}

function drawGlobe(
    context: CanvasRenderingContext2D,
    frame: CapturedFrame,
    crop: Crop,
    size: {width: number, height: number},
) {
    const source = document.createElement("canvas")
    source.width = frame.width
    source.height = frame.height
    source.getContext("2d")?.putImageData(
        new ImageData(frame.pixels, frame.width, frame.height), 0, 0)

    context.imageSmoothingEnabled = true
    context.imageSmoothingQuality = "high"
    context.drawImage(source,
        crop.x, crop.y, crop.width, crop.height,
        0, 0, size.width, size.height)
}

function drawMasthead(
    context: CanvasRenderingContext2D,
    size: {width: number, height: number},
    stats: ShareStats,
    logo: HTMLImageElement,
) {
    const unit = Math.min(size.width, size.height) / 100
    const margin = 4 * unit
    const logoSize = 9 * unit
    const middle = margin + logoSize / 2

    context.drawImage(logo, margin, margin, logoSize, logoSize)

    shadow(context, unit)
    context.fillStyle = "#FFFFFF"
    context.textBaseline = "alphabetic"

    const nameSize = 5 * unit
    context.textAlign = "left"
    context.font = `${nameSize}px ${TITLE_FONT}`
    context.fillText("ClickPlanet", margin + logoSize + 1.6 * unit, capCentred(context, middle, nameSize))

    const linkSize = 3.2 * unit
    context.textAlign = "right"
    context.font = `500 ${linkSize}px ${LABEL_FONT}`
    tracking(context, 0.1 * unit)
    context.fillText(shareLabel(stats.country.code),
        size.width - margin, capCentred(context, middle, linkSize))
    tracking(context, 0)

    clearShadow(context)
}

function drawBadge(
    context: CanvasRenderingContext2D,
    size: {width: number, height: number},
    stats: ShareStats,
    atlas: HTMLImageElement,
) {
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

    const flagRun = flag.width > 0 ? flag.width + gap : 0
    const room = size.width - margin * 2 - padding * 2 - flagRun
    context.font = `${nameSize}px ${TITLE_FONT}`
    const natural = context.measureText(stats.country.name).width
    if (natural > room) {
        nameSize *= room / natural
        context.font = `${nameSize}px ${TITLE_FONT}`
    }
    const nameWidth = context.measureText(stats.country.name).width
    const nameCap = capHeight(context, nameSize)

    const label = statsLine(stats)
    context.font = `${labelSize}px ${LABEL_FONT}`
    tracking(context, 0.18 * unit)
    const labelWidth = context.measureText(label).width
    const labelCap = capHeight(context, labelSize)
    tracking(context, 0)

    const topRow = Math.max(flag.height, nameCap)
    const width = Math.max(flagRun + nameWidth, labelWidth) + padding * 2
    const height = topRow + gap + labelCap + padding * 2
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
    context.textBaseline = "alphabetic"
    context.font = `${nameSize}px ${TITLE_FONT}`
    context.fillText(stats.country.name,
        left + padding + flagRun, capCentred(context, middle, nameSize))

    clearShadow(context)
    context.fillStyle = "#FFFFFFB3"
    context.font = `${labelSize}px ${LABEL_FONT}`
    tracking(context, 0.18 * unit)
    context.fillText(label, left + padding, top + padding + topRow + gap + labelCap)
    tracking(context, 0)
}

function capCentred(context: CanvasRenderingContext2D, middle: number, fontSize: number): number {
    return middle + capHeight(context, fontSize) / 2
}

function capHeight(context: CanvasRenderingContext2D, fontSize: number): number {
    const measured = context.measureText("H").actualBoundingBoxAscent

    return measured > 0 ? measured : fontSize * TITLE_CAP_HEIGHT
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

function tracking(context: CanvasRenderingContext2D, pixels: number) {
    (context as {letterSpacing?: string}).letterSpacing = `${pixels}px`
}

const images = new Map<string, Promise<HTMLImageElement>>()

function loadImage(url: string): Promise<HTMLImageElement> {
    const known = images.get(url)
    if (known) return known

    const loading = new Promise<HTMLImageElement>((resolve, reject) => {
        const image = new Image()
        image.onload = () => resolve(image)
        image.onerror = () => {
            images.delete(url)
            reject(new Error(`${url} could not be loaded for the share image`))
        }
        image.src = url
    })

    images.set(url, loading)
    return loading
}

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
