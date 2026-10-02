export type ShareDelivery =
    | "sheet"
    | "copy"
    | "download"

export type ShareOutcome =
    | "shared"
    | "cancelled"
    | "copied"
    | "downloaded"

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

async function copyToClipboard(file: File): Promise<boolean> {
    if (!takesImagesOnClipboard()) return false

    try {
        await navigator.clipboard.write([new ClipboardItem({[file.type]: file})])
        return true
    } catch {
        return false
    }
}

function download(file: File) {
    const url = URL.createObjectURL(file)

    const link = document.createElement("a")
    link.href = url
    link.download = file.name
    link.click()

    // Revoking in the same tick cancels the download in some browsers.
    setTimeout(() => URL.revokeObjectURL(url), 0)
}

function isAbort(error: unknown): boolean {
    return error instanceof DOMException && error.name === "AbortError"
}
