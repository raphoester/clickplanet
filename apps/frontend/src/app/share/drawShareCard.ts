import {CapturedFrame} from "../viewer/capture.ts";
import {regions} from "../viewer/atlas.ts";
import {ATLAS_URL} from "../viewer/atlasAsset.ts";
import {cardLayout, Crop, fitInBox, shareLabel, ShareStats, statsLine} from "../../domain/shareCard.ts";
import {TITLE_CAP_HEIGHT} from "../titleFont.ts";

const LOGO_URL = "/static/logo.svg"

type Look = {
    display: string
    text: string
    ink: string
    panel: string
    white: string
    soft: string
}

export async function drawShareCard(frame: CapturedFrame, stats: ShareStats): Promise<Blob> {
    const {crop, ...size} = cardLayout(frame.width, frame.height)

    const canvas = document.createElement("canvas")
    canvas.width = size.width
    canvas.height = size.height

    const context = canvas.getContext("2d")
    if (!context) throw new Error("this browser would not give a 2D context for the share image")

    drawGlobe(context, frame, crop, size)

    const look = lookOfThePage()

    // Fonts must be ready before the first measureText: a fallback face measures differently.
    const [atlas, logo] = await Promise.all([
        loadImage(ATLAS_URL),
        loadImage(LOGO_URL),
        fontsReady(look),
    ])

    drawMasthead(context, size, stats, logo, look)
    drawBadge(context, size, stats, atlas, look)

    return encode(canvas)
}

function lookOfThePage(): Look {
    const style = getComputedStyle(document.documentElement)
    const token = (name: string) => style.getPropertyValue(name).trim()

    return {
        display: token("--font-display"),
        text: token("--font-text"),
        ink: token("--ink"),
        panel: token("--panel"),
        white: token("--text"),
        soft: token("--text-soft"),
    }
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
    look: Look,
) {
    const unit = Math.min(size.width, size.height) / 100
    const margin = 4 * unit
    const logoSize = 9 * unit
    const middle = margin + logoSize / 2

    context.drawImage(logo, margin, margin, logoSize, logoSize)

    context.textBaseline = "alphabetic"

    const nameSize = 5 * unit
    context.textAlign = "left"
    context.font = `${nameSize}px ${look.display}`
    outlinedText(context, "ClickPlanet",
        margin + logoSize + 1.6 * unit, capCentred(context, middle, nameSize),
        0.7 * unit, look)

    const linkSize = 3.2 * unit
    context.textAlign = "right"
    context.font = `600 ${linkSize}px ${look.text}`
    outlinedText(context, shareLabel(stats.country.code),
        size.width - margin, capCentred(context, middle, linkSize),
        0.5 * unit, look)
}

function drawBadge(
    context: CanvasRenderingContext2D,
    size: {width: number, height: number},
    stats: ShareStats,
    atlas: HTMLImageElement,
    look: Look,
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
    context.font = `${nameSize}px ${look.display}`
    const natural = context.measureText(stats.country.name).width
    if (natural > room) {
        nameSize *= room / natural
        context.font = `${nameSize}px ${look.display}`
    }
    const nameWidth = context.measureText(stats.country.name).width
    const nameCap = capHeight(context, nameSize)

    const label = statsLine(stats)
    context.font = `600 ${labelSize}px ${look.text}`
    const labelWidth = context.measureText(label).width
    const labelCap = capHeight(context, labelSize)

    const topRow = Math.max(flag.height, nameCap)
    const width = Math.max(flagRun + nameWidth, labelWidth) + padding * 2
    const height = topRow + gap + labelCap + padding * 2
    const left = margin
    const top = size.height - margin - height

    panel(context, left, top, width, height, 2.6 * unit, unit, look)

    const middle = top + padding + topRow / 2
    if (region && flag.width > 0) {
        context.drawImage(atlas,
            region.x, region.y, region.width, region.height,
            left + padding, middle - flag.height / 2, flag.width, flag.height)
    }

    drop(context, 0.5 * unit, look)
    context.fillStyle = look.white
    context.textBaseline = "alphabetic"
    context.font = `${nameSize}px ${look.display}`
    context.fillText(stats.country.name,
        left + padding + flagRun, capCentred(context, middle, nameSize))

    clearShadow(context)
    context.fillStyle = look.soft
    context.font = `600 ${labelSize}px ${look.text}`
    context.fillText(label, left + padding, top + padding + topRow + gap + labelCap)
}

function capCentred(context: CanvasRenderingContext2D, middle: number, fontSize: number): number {
    return middle + capHeight(context, fontSize) / 2
}

function capHeight(context: CanvasRenderingContext2D, fontSize: number): number {
    const measured = context.measureText("H").actualBoundingBoxAscent

    return measured > 0 ? measured : fontSize * TITLE_CAP_HEIGHT
}

function outlinedText(
    context: CanvasRenderingContext2D,
    text: string,
    x: number, y: number,
    outline: number,
    look: Look,
) {
    context.lineJoin = "round"
    context.lineWidth = outline
    context.strokeStyle = look.ink
    drop(context, outline, look)
    context.strokeText(text, x, y)
    clearShadow(context)

    context.fillStyle = look.white
    context.fillText(text, x, y)
}

function panel(
    context: CanvasRenderingContext2D,
    x: number, y: number, width: number, height: number,
    radius: number,
    unit: number,
    look: Look,
) {
    const outline = 0.45 * unit

    roundedRect(context, x, y + 0.7 * unit, width, height, radius)
    context.fillStyle = look.ink
    context.fill()

    roundedRect(context, x, y, width, height, radius)
    context.fillStyle = look.panel
    context.fill()
    context.lineWidth = outline
    context.strokeStyle = look.ink
    context.stroke()
}

function roundedRect(
    context: CanvasRenderingContext2D,
    x: number, y: number, width: number, height: number,
    radius: number,
) {
    context.beginPath()
    if (context.roundRect) context.roundRect(x, y, width, height, radius)
    else context.rect(x, y, width, height)
}

function drop(context: CanvasRenderingContext2D, offset: number, look: Look) {
    context.shadowColor = look.ink
    context.shadowOffsetX = 0
    context.shadowOffsetY = offset
    context.shadowBlur = 0
}

function clearShadow(context: CanvasRenderingContext2D) {
    context.shadowColor = "transparent"
    context.shadowOffsetX = 0
    context.shadowOffsetY = 0
    context.shadowBlur = 0
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

async function fontsReady(look: Look): Promise<void> {
    await Promise.all([`16px ${look.display}`, `600 16px ${look.text}`]
        .map((font) => document.fonts?.load(font)))
    await document.fonts?.ready
}

function encode(canvas: HTMLCanvasElement): Promise<Blob> {
    return new Promise((resolve, reject) => {
        canvas.toBlob(
            (blob) => blob ? resolve(blob) : reject(new Error("the share image could not be encoded")),
            "image/png")
    })
}
