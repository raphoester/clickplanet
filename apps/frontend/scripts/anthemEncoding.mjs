import {execFileSync} from "node:child_process"
import fs from "node:fs"
import path from "node:path"

export const CACHE = "node_modules/.cache/anthems"

// Silence trimmed at both ends, so a loop has no gap, and every anthem as loud as the next.
const FILTER = [
    "silenceremove=start_periods=1:start_threshold=-50dB",
    "areverse",
    "silenceremove=start_periods=1:start_threshold=-50dB",
    "areverse",
    "loudnorm=I=-18:TP=-2:LRA=11",
].join(",")

export function encode(input) {
    const tmp = path.join(CACHE, "encoding.m4a")
    execFileSync("ffmpeg", [
        "-v", "error", "-y", "-i", input,
        "-af", FILTER, "-ar", "44100", "-ac", "1",
        "-c:a", "aac", "-b:a", "56k",
        "-map_metadata", "-1", "-fflags", "+bitexact", "-flags:a", "+bitexact",
        "-movflags", "+faststart",
        tmp,
    ])
    return fs.readFileSync(tmp)
}
