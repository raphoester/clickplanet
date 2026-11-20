import {execFileSync} from "node:child_process"
import {rmSync, writeFileSync} from "node:fs"
import {join} from "node:path"

// Each photo of the folder for seconds, then the next, at 30 frames a second, with a silent track: YouTube plays a Short
// with no sound, and some players refuse a file with no audio stream.
export function slideshow(folder, photos, seconds, out) {
    const list = join(folder, "slideshow.txt")
    const lines = photos.flatMap((photo) => [`file '${photo}'`, `duration ${seconds}`])
    // The concat demuxer takes the last duration from the next entry: the last photo is named once more.
    lines.push(`file '${photos.at(-1)}'`)
    writeFileSync(list, lines.join("\n") + "\n")
    try {
        execFileSync("ffmpeg", [
            "-y", "-loglevel", "error",
            "-f", "concat", "-safe", "0", "-i", list,
            "-f", "lavfi", "-i", "anullsrc=r=44100:cl=stereo",
            "-shortest",
            // ffmpeg 9 shows the last photo, named twice, for twice its time.
            "-t", String(photos.length * seconds),
            "-vf", "fps=30,format=yuv420p",
            "-c:v", "libx264", "-preset", "veryfast", "-crf", "20",
            "-c:a", "aac", "-b:a", "64k",
            "-movflags", "+faststart",
            out,
        ], {stdio: ["ignore", "inherit", "inherit"]})
    } finally {
        rmSync(list, {force: true})
    }
}
