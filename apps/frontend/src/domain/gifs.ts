import type {Gif} from "../backends/chat.ts";

export type Size = {
    width: number
    height: number
}

// No bigger than it is nor than the box, in its own shape: the line holds its place before the image comes.
export function fitted(gif: Gif, box: Size): Size {
    const scale = Math.min(1, box.width / gif.width, box.height / gif.height)
    return {width: Math.round(gif.width * scale), height: Math.round(gif.height * scale)}
}
