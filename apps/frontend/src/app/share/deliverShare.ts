/**
 * Getting the finished PNG out of the page, by whichever of the three ways this
 * browser actually offers.
 *
 * The order is how far each one gets the image with no further work from the
 * player: the share sheet posts it, the clipboard needs one paste, a download
 * needs the file to be found again. Each rung is tried and the next is only
 * reached when the browser says no — a Safari on a desktop has `navigator.share`
 * but will not take files, and no amount of feature detection on `share` alone
 * would have caught that.
 */

export type ShareOutcome =
    | "shared"
    /** The player opened the share sheet and closed it again. Nothing was
     *  delivered, and nothing should be: a fallback here would put an image on
     *  their clipboard seconds after they said no. */
    | "cancelled"
    | "copied"
    | "downloaded"

export async function deliverShare(file: File, text: string): Promise<ShareOutcome> {
    const sheet = await offerToShareSheet(file, text)
    if (sheet) return sheet

    if (await copyToClipboard(file, text)) return "copied"

    download(file)
    return "downloaded"
}

async function offerToShareSheet(file: File, text: string): Promise<ShareOutcome | undefined> {
    const data: ShareData = {files: [file], title: "ClickPlanet", text}

    // `canShare` with the files in hand is the only honest test: the browsers
    // that share text but not files answer false here and true for `share`.
    if (!navigator.share || !navigator.canShare?.(data)) return undefined

    try {
        await navigator.share(data)
        return "shared"
    } catch (error) {
        if (isAbort(error)) return "cancelled"
        return undefined
    }
}

async function copyToClipboard(file: File, text: string): Promise<boolean> {
    if (!navigator.clipboard?.write || typeof ClipboardItem === "undefined") return false

    const image = {[file.type]: file}
    const withText = {...image, "text/plain": new Blob([text], {type: "text/plain"})}

    // Two types in one item is what puts the link on the clipboard alongside the
    // picture, where the paste target takes text. Where it does not, the item is
    // refused whole — so the image goes on its own rather than nothing at all.
    return await written(withText) || await written(image)
}

async function written(types: Record<string, Blob>): Promise<boolean> {
    try {
        await navigator.clipboard.write([new ClipboardItem(types)])
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

    // Revoking in the same tick cancels the download in some browsers: the link
    // has been clicked, not yet read.
    setTimeout(() => URL.revokeObjectURL(url), 0)
}

function isAbort(error: unknown): boolean {
    return error instanceof DOMException && error.name === "AbortError"
}
