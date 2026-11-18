// ?diag: a phone has no console, so a page left blank shows its errors and its storage on screen (docs/diag.md).

const SHOW_AFTER_MS = 10_000
const VALUE_CHARS = 160
// A screenshot of the report is sent around: a token's value stays off it.
const SECRET = /session|token|account/i

const seen: string[] = []

if (new URLSearchParams(window.location.search).has("diag")) {
    window.addEventListener("error", (e) => note(`error: ${e.message} @ ${e.filename}:${e.lineno}:${e.colno}`))
    window.addEventListener("unhandledrejection", (e) => note(`rejection: ${describe(e.reason)}`))
    const consoleError = console.error.bind(console)
    console.error = (...args: unknown[]) => {
        note(`console: ${args.map(describe).join(" ")}`)
        consoleError(...args)
    }
    setTimeout(showReport, SHOW_AFTER_MS)
}

function note(line: string) {
    seen.push(`${Math.round(performance.now())}ms ${line}`)
}

function describe(value: unknown): string {
    if (value instanceof Error) return `${value.name}: ${value.message}\n${value.stack ?? ""}`
    try {
        return typeof value === "string" ? value : JSON.stringify(value)
    } catch {
        return String(value)
    }
}

function storageLines(name: string, storage: () => Storage): string[] {
    try {
        const kept = storage()
        const lines = [`${name}: ${kept.length} keys`]
        for (let i = 0; i < kept.length; i++) {
            const key = kept.key(i) ?? ""
            const value = kept.getItem(key) ?? ""
            const shown = SECRET.test(key) ? "(hidden)" : value.slice(0, VALUE_CHARS)
            lines.push(`  ${key} (${value.length}): ${shown}`)
        }
        return lines
    } catch (e) {
        return [`${name}: unreadable, ${describe(e)}`]
    }
}

function canvasLine(): string {
    const canvas = document.querySelector("canvas")
    if (!canvas) return "canvas: none"
    const gl = canvas.getContext("webgl2") ?? canvas.getContext("webgl")
    return `canvas: ${canvas.width}x${canvas.height}, css ${canvas.clientWidth}x${canvas.clientHeight}, lost ${gl?.isContextLost()}`
}

function showReport() {
    const report = [
        `url: ${window.location.href}`,
        `agent: ${navigator.userAgent}`,
        `screen: ${window.innerWidth}x${window.innerHeight} @${window.devicePixelRatio}`,
        `root children: ${document.getElementById("root")?.children.length}`,
        canvasLine(),
        `errors (${seen.length}):`,
        ...seen,
        ...storageLines("localStorage", () => window.localStorage),
        ...storageLines("sessionStorage", () => window.sessionStorage),
    ].join("\n")

    const panel = document.createElement("pre")
    panel.textContent = report
    Object.assign(panel.style, {
        position: "fixed", inset: "0", zIndex: "2147483647", margin: "0", padding: "12px", overflow: "auto",
        background: "#fff", color: "#000", font: "11px/1.35 monospace", whiteSpace: "pre-wrap", wordBreak: "break-all",
    })
    panel.addEventListener("click", () => panel.remove())
    document.body.appendChild(panel)
}
