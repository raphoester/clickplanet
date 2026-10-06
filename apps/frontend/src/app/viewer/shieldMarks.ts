import * as THREE from "three"

const CELL = 128
const COLUMNS = 4

const SHIELD = {
    size: 48,
    body: "M24 5 L39 10 V22 C39 32 32 39 24 43 C16 39 9 32 9 22 V10 Z",
    height: 38,
} as const

const SHIELD_HEIGHT = 0.8
const OUTLINE = 0.05
const DROP = 0.035

const COUNT = {size: 0.48, width: 0.41, rise: 0.03, outline: 0.075}

type Look = {
    display: string
    ink: string
    steel: string
    steelLight: string
    digit: string
}

export function shieldCellsOf(most: number): THREE.Vector2 {
    return new THREE.Vector2(COLUMNS, Math.ceil((Math.max(1, most) + 1) / COLUMNS))
}

export function drawShieldMarks(most: number, onRedrawn: () => void): THREE.CanvasTexture {
    const cells = shieldCellsOf(most)

    const canvas = document.createElement("canvas")
    canvas.width = cells.x * CELL
    canvas.height = cells.y * CELL

    const context = canvas.getContext("2d")
    if (!context) throw new Error("failed to get a 2d context for the shield marks")

    const look = lookOfThePage()
    const texture = new THREE.CanvasTexture(canvas)

    const draw = () => {
        context.clearRect(0, 0, canvas.width, canvas.height)
        for (let count = 0; count <= most; count++) {
            context.save()
            context.translate((count % COLUMNS) * CELL, Math.floor(count / COLUMNS) * CELL)
            drawShield(context, look)
            if (count > 0) drawCount(context, count, look)
            context.restore()
        }
        texture.needsUpdate = true
    }

    draw()
    document.fonts?.load(`${CELL}px ${look.display}`).then(() => {
        draw()
        onRedrawn()
    }, () => {})

    return texture
}

function drawShield(context: CanvasRenderingContext2D, look: Look) {
    const scale = CELL * SHIELD_HEIGHT / SHIELD.height
    const body = new Path2D(SHIELD.body)

    context.save()
    context.translate(CELL / 2 - SHIELD.size / 2 * scale, CELL / 2 - SHIELD.size / 2 * scale)
    context.scale(scale, scale)
    context.lineJoin = "round"
    context.lineWidth = CELL * OUTLINE / scale

    context.save()
    context.translate(0, CELL * DROP / scale)
    context.fillStyle = look.ink
    context.strokeStyle = look.ink
    context.fill(body)
    context.stroke(body)
    context.restore()

    context.fillStyle = look.steelLight
    context.fill(body)

    context.save()
    context.clip(body)
    context.fillStyle = look.steel
    context.fillRect(SHIELD.size / 2, 0, SHIELD.size / 2, SHIELD.size)
    context.restore()

    context.strokeStyle = look.ink
    context.stroke(body)
    context.restore()
}

function drawCount(context: CanvasRenderingContext2D, count: number, look: Look) {
    const text = String(count)
    context.font = `${CELL * COUNT.size}px ${look.display}`
    context.textAlign = "left"
    context.textBaseline = "alphabetic"
    context.lineJoin = "round"

    const measured = context.measureText(text)
    const width = measured.actualBoundingBoxLeft + measured.actualBoundingBoxRight
    const height = measured.actualBoundingBoxAscent + measured.actualBoundingBoxDescent
    const fit = Math.min(1, CELL * COUNT.width / width)

    context.save()
    context.translate(CELL / 2, CELL * (0.5 - COUNT.rise))
    context.scale(fit, fit)

    const left = measured.actualBoundingBoxLeft - width / 2
    const baseline = height / 2 - measured.actualBoundingBoxDescent
    const drop = CELL * DROP / fit

    context.lineWidth = CELL * COUNT.outline / fit
    context.strokeStyle = look.ink
    context.fillStyle = look.ink
    context.strokeText(text, left, baseline + drop)
    context.fillText(text, left, baseline + drop)
    context.strokeText(text, left, baseline)

    context.fillStyle = look.digit
    context.fillText(text, left, baseline)
    context.restore()
}

function lookOfThePage(): Look {
    const style = getComputedStyle(document.documentElement)
    const token = (name: string) => style.getPropertyValue(name).trim()

    return {
        display: token("--font-display"),
        ink: token("--ink"),
        steel: token("--shield"),
        steelLight: token("--shield-light"),
        digit: token("--text"),
    }
}
