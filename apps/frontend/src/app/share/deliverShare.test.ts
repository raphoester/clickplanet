// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {deliverShare} from "./deliverShare.ts"

const image = () => new File([new Uint8Array([1, 2, 3])], "clickplanet-fr.png", {type: "image/png"})
const TEXT = "France is #2 on ClickPlanet. https://clickplanet.lol/?c=fr"

type Navigatorish = {
    share?: unknown
    canShare?: unknown
    clipboard?: unknown
}

function browser(capabilities: Navigatorish) {
    for (const key of ["share", "canShare", "clipboard"] as const) {
        Object.defineProperty(navigator, key, {value: capabilities[key], configurable: true, writable: true})
    }
}

/** jsdom has no `matchMedia` at all, so the ladder reads as a desktop unless a
 *  test says otherwise — which is the safe default for it to have. */
function touchDevice(coarse: boolean) {
    vi.stubGlobal("matchMedia", (query: string) =>
        ({matches: coarse && query === "(pointer: coarse)"}))
}

/** jsdom carries neither, and what the ladder does with them is the whole point. */
function clipboard(write = vi.fn().mockResolvedValue(undefined)) {
    vi.stubGlobal("ClipboardItem", class {
        types: string[]
        constructor(public items: Record<string, Blob>) {
            this.types = Object.keys(items)
        }
    })
    return write
}

/** jsdom carries neither either, and they are left in place rather than
 *  restored: the revoke is deliberately one tick late, so a stub taken away at
 *  the end of the test is a stub the download outlives. */
function downloads() {
    URL.createObjectURL = vi.fn().mockReturnValue("blob:fake")
    URL.revokeObjectURL = vi.fn()
    return vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {})
}

afterEach(() => {
    browser({})
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
})

describe("delivering the share image", () => {
    it("hands the file to the share sheet where the browser will take one", async () => {
        touchDevice(true)
        const share = vi.fn().mockResolvedValue(undefined)
        browser({share, canShare: () => true, clipboard: {write: vi.fn()}})

        expect(await deliverShare(image(), TEXT)).toBe("shared")
        expect(share.mock.calls[0][0]).toMatchObject({text: TEXT})
        expect(share.mock.calls[0][0].files).toHaveLength(1)
    })

    // A desktop Safari has `navigator.share` and refuses files, so `share`
    // alone is not a test of anything.
    it("asks whether the sheet takes files rather than whether it exists", async () => {
        touchDevice(true)
        const share = vi.fn()
        const write = clipboard()
        browser({share, canShare: () => false, clipboard: {write}})

        expect(await deliverShare(image(), TEXT)).toBe("copied")
        expect(share).not.toHaveBeenCalled()
    })

    it("stops when the player closes the sheet, instead of copying behind their back", async () => {
        touchDevice(true)
        const write = clipboard()
        browser({
            share: vi.fn().mockRejectedValue(new DOMException("cancelled", "AbortError")),
            canShare: () => true,
            clipboard: {write},
        })

        expect(await deliverShare(image(), TEXT)).toBe("cancelled")
        expect(write).not.toHaveBeenCalled()
    })

    it("falls to the clipboard when the sheet itself fails", async () => {
        touchDevice(true)
        const write = clipboard()
        browser({
            share: vi.fn().mockRejectedValue(new Error("the sheet fell over")),
            canShare: () => true,
            clipboard: {write},
        })

        expect(await deliverShare(image(), TEXT)).toBe("copied")
    })

    // Chrome on macOS answers `canShare({files})` true and then hands Telegram
    // the sentence without the picture. A rung that reports success and loses
    // what it was given is worse than not having it.
    it("keeps a desktop off the share sheet, whatever it claims it can share", async () => {
        touchDevice(false)
        const share = vi.fn()
        const write = clipboard()
        browser({share, canShare: () => true, clipboard: {write}})

        expect(await deliverShare(image(), TEXT)).toBe("copied")
        expect(share).not.toHaveBeenCalled()
    })

    // The link is drawn into the image for this reason. Handed a clipboard
    // carrying the picture and the sentence, a chat window pastes the sentence —
    // the same complaint the desktop share sheet produced.
    it("copies the picture alone, with no text for a paste target to prefer", async () => {
        const write = clipboard()
        browser({clipboard: {write}})

        expect(await deliverShare(image(), TEXT)).toBe("copied")
        expect(write.mock.calls[0][0][0].types).toEqual(["image/png"])
    })

    it("downloads the file when there is no clipboard to write to", async () => {
        const click = downloads()
        browser({})

        expect(await deliverShare(image(), TEXT)).toBe("downloaded")
        expect(click).toHaveBeenCalledTimes(1)
    })

    it("downloads it when the clipboard refuses it outright", async () => {
        const click = downloads()
        clipboard()
        browser({clipboard: {write: vi.fn().mockRejectedValue(new Error("denied"))}})

        expect(await deliverShare(image(), TEXT)).toBe("downloaded")
        expect(click).toHaveBeenCalledTimes(1)
    })
})
