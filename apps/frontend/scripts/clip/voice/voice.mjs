// The war report's bench, voiced (docs/video-lab.md, "The war report's bench"): each script of a bench.json said by
// each voice asked, one sentence at a time, as <out>/<id>-<voice>.wav, and a voices.json of what was made, with when
// each line is said, for the subtitles.
//
//   node voice.mjs <bench.json> <out folder> [am_adam,af_heart] [speed]
//
// Kokoro truncates a long input, so a script is said a line at a time, each line's silent edges cut, and the lines
// joined with a short breath between.
import {mkdir, readFile, writeFile} from "node:fs/promises"
import path from "node:path"
import {KokoroTTS} from "kokoro-js"
import {joined, trimmed, wavOf} from "./wav.mjs"

const MODEL = "onnx-community/Kokoro-82M-v1.0-ONNX"
// The owner's pick, tried by hand on 2026-11-20.
const DEFAULT_VOICE = "am_adam"
// The owner's ear (2026-11-23): Adam at Kokoro's own pace was too slow, 1.5 times too fast, 1.15 right, picked from
// 1.1 to 1.25; the 1.25 s Kokoro's edges and a 0.3 s breath left between two sentences far too long.
const DEFAULT_SPEED = 1.15
const BREATH_SECONDS = 0.25

const [benchFile, out, voicesArg, speedArg] = process.argv.slice(2)
if (!benchFile || !out) {
    console.error("usage: node voice.mjs <bench.json> <out folder> [am_adam,af_heart] [speed]")
    process.exit(2)
}
const voices = (voicesArg || DEFAULT_VOICE).split(",").map((voice) => voice.trim()).filter(Boolean)
const speed = speedArg ? Number(speedArg) : DEFAULT_SPEED
if (!(speed > 0.5 && speed < 2)) throw new Error(`a speed is between 0.5 and 2, not ${speedArg}`)

const bench = JSON.parse(await readFile(benchFile, "utf8"))
await mkdir(out, {recursive: true})
const tts = await KokoroTTS.from_pretrained(MODEL, {dtype: "fp32", device: "cpu"})
for (const voice of voices) {
    if (!(voice in tts.voices)) throw new Error(`Kokoro has no voice ${voice}: ${Object.keys(tts.voices).join(", ")}`)
}

const tenths = (seconds) => Math.round(seconds * 100) / 100
const made = []
for (const entry of bench.entries) {
    if (entry.failed || !entry.lines?.length) continue
    for (const voice of voices) {
        const parts = []
        let rate = 24000
        for (const line of entry.lines) {
            const audio = await tts.generate(line, {voice, speed})
            parts.push(trimmed(audio.audio, audio.sampling_rate))
            rate = audio.sampling_rate
        }
        const samples = joined(parts, rate, BREATH_SECONDS)
        let at = 0
        const lines = entry.lines.map((text, i) => {
            const from = at
            at += parts[i].length / rate
            const line = {text, from: tenths(from), to: tenths(at)}
            at += BREATH_SECONDS
            return line
        })
        const file = `${entry.id}-${voice}.wav`
        await writeFile(path.join(out, file), wavOf(samples, rate))
        made.push({id: entry.id, voice, speed, file, seconds: tenths(samples.length / rate), lines})
        console.log(`${file}: ${(samples.length / rate).toFixed(1)} s at ${speed}`)
    }
}
await writeFile(path.join(out, "voices.json"), JSON.stringify({voices, speed, made}, null, 2) + "\n")
