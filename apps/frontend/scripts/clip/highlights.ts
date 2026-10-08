import {execFile} from "node:child_process"
import {writeFileSync} from "node:fs"
import {promisify} from "node:util"
import {ANTHEMS} from "../../src/app/anthem/anthemsAsset.ts"
import {CLIP_ANTHEMS} from "../../src/clip/clipAnthemsAsset.ts"
import {highlightsOf, soundsOf} from "../../src/domain/clip/highlight.ts"

const ASSET = "src/clip/anthemHighlightsAsset.ts"
const RATE = 22050
const AT_ONCE = 8

const run = promisify(execFile)

async function decoded(url: string): Promise<Float32Array> {
    const {stdout} = await run("ffmpeg", ["-loglevel", "error", "-i", url.slice(1), "-f", "f32le", "-ac", "1", "-ar", String(RATE), "-"],
        {encoding: "buffer", maxBuffer: 1 << 30})
    return new Float32Array(stdout.buffer, stdout.byteOffset, stdout.byteLength / 4)
}

const queue = [...new Set([...Object.values(ANTHEMS), ...Object.values(CLIP_ANTHEMS)].map((anthem) => anthem.url))].sort()
const lines = new Map<string, string>()
await Promise.all(Array.from({length: AT_ONCE}, async () => {
    for (let url = queue.pop(); url; url = queue.pop()) {
        const samples = await decoded(url)
        const at = highlightsOf(soundsOf(samples, RATE))
        const seconds = samples.length / RATE
        lines.set(url, `    ${JSON.stringify(url)}: {seconds: ${seconds.toFixed(2)}, at: [${at.map((moment) => moment.toFixed(2)).join(", ")}]},`)
        console.log(`${url}  ${seconds.toFixed(1)}s  ${at.map((moment) => moment.toFixed(1)).join(" ") || "none"}`)
    }
}))

writeFileSync(ASSET, [
    "export const HIGHLIGHTS: Readonly<Record<string, {seconds: number, at: readonly number[]}>> = {",
    ...[...lines.keys()].sort().map((url) => lines.get(url)),
    "}",
    "",
].join("\n"))
console.log(`${lines.size} recordings`)
