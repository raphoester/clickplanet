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
 *
 * **The share sheet is offered on a touch device only**, which is the one thing
 * here that is not a feature test. On a phone the sheet *is* how you share, and
 * it carries the file. On a desktop it is a shim over the OS share services,
 * and `canShare({files})` answers true for services that then keep the text and
 * drop the image — measured with Chrome on macOS into Telegram, which posted
 * the sentence and no picture. A rung that reports success and silently loses
 * the thing being shared is worse than not having it: the clipboard puts a real
 * PNG into that same Telegram window with one paste.
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
    if (sharesFilesFaithfully()) {
        const sheet = await offerToShareSheet(file, text)
        if (sheet) return sheet
    }

    if (await copyToClipboard(file)) return "copied"

    download(file)
    return "downloaded"
}

/** A coarse primary pointer is a phone or a tablet, where the share sheet is the
 *  platform's own way of sending a file somewhere rather than a bridge to it. */
function sharesFilesFaithfully(): boolean {
    return window.matchMedia?.("(pointer: coarse)").matches ?? false
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

/**
 * The picture and nothing else.
 *
 * Putting the link on as `text/plain` beside it looks like a free extra and is
 * the same bug as the desktop share sheet wearing a different hat: handed a
 * clipboard carrying both, a chat window pastes the sentence. The link is drawn
 * into the image for exactly this reason, so there is nothing to lose by
 * leaving it out here and one way to be misunderstood fewer.
 */
async function copyToClipboard(file: File): Promise<boolean> {
    if (!navigator.clipboard?.write || typeof ClipboardItem === "undefined") return false

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

    // Revoking in the same tick cancels the download in some browsers: the link
    // has been clicked, not yet read.
    setTimeout(() => URL.revokeObjectURL(url), 0)
}

function isAbort(error: unknown): boolean {
    return error instanceof DOMException && error.name === "AbortError"
}
