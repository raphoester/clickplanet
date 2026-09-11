/**
 * Getting the finished PNG out of the page.
 *
 * **The player picks, rather than a ladder picking for them.** An earlier
 * version tried the three ways in turn and reported whichever answered first,
 * and both halves of that went wrong in the first minute of real use: the
 * desktop share sheet reported success and posted the text without the picture,
 * and a clipboard write the browser had quietly refused came back as a
 * download. What is offered here is only ever what this browser can actually
 * do, and what happens is only ever what was asked for.
 *
 * **The share sheet is a phone's button**, and that is the one judgement here
 * that is not a feature test. On a phone the sheet *is* how you send a file
 * somewhere and it carries one. On a desktop it is a shim over the OS share
 * services, and `canShare({files})` answers true for services that then keep
 * the text and drop the image — measured with Chrome on macOS into Telegram,
 * which posted the sentence and no picture. A desktop gets the two buttons that
 * cannot lie about what they did instead.
 */

export type ShareDelivery =
    /** The platform's own share sheet, with the file attached. */
    | "sheet"
    | "copy"
    | "download"

export type ShareOutcome =
    | "shared"
    /** The player opened the share sheet and closed it again. Nothing was
     *  delivered, and nothing should be: a fallback here would hand them a file
     *  seconds after they said no. */
    | "cancelled"
    | "copied"
    | "downloaded"

/**
 * The ways out this browser has, in the order they are put on screen.
 *
 * Read once when the menu mounts — none of it changes while the page is open —
 * and never empty: a download is always possible.
 */
export function deliveriesOffered(): ShareDelivery[] {
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

            // This is the only button a phone is shown, so a sheet that falls
            // over still has to leave the player holding the picture.
            download(file)
            return "downloaded"
        }
        case "copy":
            // No quiet fallback to the download: that is its own button an inch
            // away, and a copy that turns into a file in Downloads is exactly
            // the kind of surprise the two buttons exist to stop.
            if (!await copyToClipboard(file)) {
                throw new Error("the clipboard would not take the image")
            }
            return "copied"
        case "download":
            download(file)
            return "downloaded"
    }
}

/** A coarse primary pointer: a phone or a tablet. */
function isPhone(): boolean {
    return window.matchMedia?.("(pointer: coarse)").matches ?? false
}

/**
 * Whether the share sheet will take a PNG at all, asked with a stand-in file —
 * `canShare` judges the *kind* of thing it is given, not the bytes.
 *
 * Asking about files is the only honest test: the browsers that share text but
 * not files answer false here and true for `navigator.share`.
 */
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

/**
 * The picture and nothing else.
 *
 * Putting the link on as `text/plain` beside it looks like a free extra and is
 * the desktop share sheet's fault wearing a different hat: handed a clipboard
 * carrying both, a chat window pastes the sentence. The link is drawn into the
 * image for exactly this reason, so there is nothing to lose by leaving it out
 * here and one way to be misunderstood fewer.
 */
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

    // Revoking in the same tick cancels the download in some browsers: the link
    // has been clicked, not yet read.
    setTimeout(() => URL.revokeObjectURL(url), 0)
}

function isAbort(error: unknown): boolean {
    return error instanceof DOMException && error.name === "AbortError"
}
