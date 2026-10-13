import {download} from "../download.ts"

export type ShareDelivery =
    | "sheet"
    | "copy"
    | "download"

export type ShareOutcome =
    | "shared"
    | "cancelled"
    | "copied"
    | "downloaded"

export type LinkDelivery = "sheet" | "copy"

export type LinkOutcome = Exclude<ShareOutcome, "downloaded">

export function deliveriesOffered(): ShareDelivery[] {
    // Desktop share sheets pass canShare({files}) and then some targets drop the image.
    if (isPhone() && takesFilesThroughShareSheet()) return ["sheet"]

    return takesImagesOnClipboard() ? ["copy", "download"] : ["download"]
}

export async function deliverShare(
    delivery: ShareDelivery,
    file: File,
    text: string,
): Promise<ShareOutcome> {
    switch (delivery) {
        case "sheet": {
            const outcome = await offerToShareSheet(file, text)
            if (outcome) return outcome

            download(file)
            return "downloaded"
        }
        case "copy":
            if (!await copyToClipboard(file)) {
                throw new Error("the clipboard would not take the image")
            }
            return "copied"
        case "download":
            download(file)
            return "downloaded"
    }
}

export function linkDeliveryOffered(): LinkDelivery {
    return isPhone() && !!navigator.share ? "sheet" : "copy"
}

export async function deliverLink(delivery: LinkDelivery, url: string, text: string): Promise<LinkOutcome> {
    switch (delivery) {
        case "sheet":
            return await offerLinkToShareSheet(url, text) ?? copyLink(url)
        case "copy":
            return copyLink(url)
    }
}

function isPhone(): boolean {
    return window.matchMedia?.("(pointer: coarse)").matches ?? false
}

function takesFilesThroughShareSheet(): boolean {
    if (!navigator.share || !navigator.canShare) return false

    const probe = new File([], "clickplanet.png", {type: "image/png"})
    return navigator.canShare({files: [probe]})
}

function takesImagesOnClipboard(): boolean {
    return !!navigator.clipboard?.write && typeof ClipboardItem !== "undefined"
}

async function offerToShareSheet(file: File, text: string): Promise<ShareOutcome | undefined> {
    const data: ShareData = {files: [file], title: "ClickPlanet", text}
    if (!navigator.canShare?.(data)) return undefined

    try {
        await navigator.share(data)
        return "shared"
    } catch (error) {
        if (isAbort(error)) return "cancelled"
        return undefined
    }
}

async function offerLinkToShareSheet(url: string, text: string): Promise<LinkOutcome | undefined> {
    try {
        await navigator.share({title: "ClickPlanet", text, url})
        return "shared"
    } catch (error) {
        if (isAbort(error)) return "cancelled"
        return undefined
    }
}

async function copyLink(url: string): Promise<LinkOutcome> {
    if (!await writeToClipboard(url)) {
        throw new Error("the clipboard would not take the link")
    }
    return "copied"
}

async function writeToClipboard(text: string): Promise<boolean> {
    try {
        await navigator.clipboard.writeText(text)
        return true
    } catch {
        // An in-app browser often refuses the clipboard API and still copies a selection.
        return copySelected(text)
    }
}

function copySelected(text: string): boolean {
    const box = document.createElement("textarea")
    box.value = text
    box.readOnly = true
    box.style.position = "fixed"
    box.style.opacity = "0"
    document.body.appendChild(box)
    box.select()
    try {
        return document.execCommand("copy")
    } catch {
        return false
    } finally {
        box.remove()
    }
}

async function copyToClipboard(file: File): Promise<boolean> {
    if (!takesImagesOnClipboard()) return false

    try {
        await navigator.clipboard.write([new ClipboardItem({[file.type]: file})])
        return true
    } catch {
        return false
    }
}

function isAbort(error: unknown): boolean {
    return error instanceof DOMException && error.name === "AbortError"
}
