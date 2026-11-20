// The war report's bench, voiced (docs/video-lab.md, "The war report's bench"): each script of a bench.json said by
// each voice asked, one sentence at a time, as <out>/<id>-<voice>.wav, and a voices.json of what was made.
//
//   node voice.mjs <bench.json> <out folder> [am_adam,af_heart]
//
// Kokoro truncates a long input, so a script is said a line at a time and the lines joined with a breath between.
import {mkdir, readFile, writeFile} from "node:fs/promises"
import path from "node:path"
import {KokoroTTS} from "kokoro-js"
import {joined, wavOf} from "./wav.mjs"

const MODEL = "onnx-community/Kokoro-82M-v1.0-ONNX"
// The owner's pick, tried by hand on 2026-11-20.
const DEFAULT_VOICE = "am_adam"
const BREATH_SECONDS = 0.3

const [benchFile, out, voicesArg] = process.argv.slice(2)
if (!benchFile || !out) {
    console.error("usage: node voice.mjs <bench.json> <out folder> [am_adam,af_heart]")
    process.exit(2)
}
const voices = (voicesArg || DEFAULT_VOICE).split(",").map((voice) => voice.trim()).filter(Boolean)

const bench = JSON.parse(await readFile(benchFile, "utf8"))
await mkdir(out, {recursive: true})
const tts = await KokoroTTS.from_pretrained(MODEL, {dtype: "fp32", device: "cpu"})
for (const voice of voices) {
    if (!(voice in tts.voices)) throw new Error(`Kokoro has no voice ${voice}: ${Object.keys(tts.voices).join(", ")}`)
}

const made = []
for (const entry of bench.entries) {
    if (entry.failed || !entry.lines?.length) continue
    for (const voice of voices) {
        const parts = []
        let rate = 24000
        for (const line of entry.lines) {
            const audio = await tts.generate(line, {voice})
            parts.push(audio.audio)
            rate = audio.sampling_rate
        }
        const samples = joined(parts, rate, BREATH_SECONDS)
        const file = `${entry.id}-${voice}.wav`
        await writeFile(path.join(out, file), wavOf(samples, rate))
        made.push({id: entry.id, voice, file, seconds: Math.round(samples.length / rate * 10) / 10})
        console.log(`${file}: ${(samples.length / rate).toFixed(1)} s`)
    }
}
await writeFile(path.join(out, "voices.json"), JSON.stringify({voices, made}, null, 2) + "\n")
