import {AttachmentRules} from "../backends/support.ts"

export type Refusal =
    | {reason: "too-many", most: number}
    | {reason: "too-big", name: string, mostBytes: number}
    | {reason: "not-media", name: string}

export type Choice = {
    files: File[]
    refusal?: Refusal
}

// The files chosen so far and the ones just picked, as far as the rules take them: the first one refused says why.
export function withPicked(chosen: File[], picked: File[], rules: AttachmentRules): Choice {
    const files = [...chosen]
    let refusal: Refusal | undefined

    for (const file of picked) {
        if (!rules.contentTypes.includes(file.type) || file.size === 0) {
            refusal ??= {reason: "not-media", name: file.name}
        } else if (file.size > rules.maxBytes) {
            refusal ??= {reason: "too-big", name: file.name, mostBytes: rules.maxBytes}
        } else if (files.length >= rules.maxPerNote) {
            refusal ??= {reason: "too-many", most: rules.maxPerNote}
        } else {
            files.push(file)
        }
    }
    return {files, refusal}
}

export function isPhoto(file: Blob): boolean {
    return file.type.startsWith("image/")
}
