// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {deliveriesOffered, deliverShare} from "./deliverShare.ts"

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

/** jsdom has no `matchMedia` at all, so this reads as a desktop unless a test
 *  says otherwise — which is the safe default for it to have. */
function phone(coarse: boolean) {
    vi.stubGlobal("matchMedia", (query: string) =>
        ({matches: coarse && query === "(pointer: coarse)"}))
}

/** jsdom carries neither, and what the delivery does with them is the point. */
function clipboard(write = vi.fn().mockResolvedValue(undefined)) {
    vi.stubGlobal("ClipboardItem", class {
        types: string[]
        constructor(public items: Record<string, Blob>) {
            this.types = Object.keys(items)
        }
    })
    return write
}

/** They are left in place rather than restored: the revoke is deliberately one
 *  tick late, so a stub taken away at the end of the test is one the download
 *  outlives. */
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

describe("what a browser is offered", () => {
    it("gives a phone its share sheet, and nothing else to think about", () => {
        phone(true)
        browser({share: vi.fn(), canShare: () => true, clipboard: {write: vi.fn()}})
        clipboard()

        expect(deliveriesOffered()).toEqual(["sheet"])
    })

    // Chrome on macOS answers `canShare({files})` true and then hands Telegram
    // the sentence without the picture. A button that reports success and loses
    // what it was given is worse than not having it.
    it("keeps a desktop off the share sheet, whatever it claims it can share", () => {
        phone(false)
        browser({share: vi.fn(), canShare: () => true, clipboard: {write: vi.fn()}})
        clipboard()

        expect(deliveriesOffered()).toEqual(["copy", "download"])
    })

    // A desktop Safari has `navigator.share` and refuses files, so `share`
    // alone is not a test of anything.
    it("asks whether the sheet takes files rather than whether it exists", () => {
        phone(true)
        browser({share: vi.fn(), canShare: () => false, clipboard: {write: vi.fn()}})
        clipboard()

        expect(deliveriesOffered()).toEqual(["copy", "download"])
    })

    it("drops the copy button where there is no clipboard to write to", () => {
        phone(false)
        browser({})

        expect(deliveriesOffered()).toEqual(["download"])
    })

    it("always offers something: a download needs nothing of the browser", () => {
        phone(true)
        browser({})

        expect(deliveriesOffered()).toEqual(["download"])
    })
})

describe("delivering the share image", () => {
    it("hands the file and the text to the share sheet", async () => {
        const share = vi.fn().mockResolvedValue(undefined)
        browser({share, canShare: () => true})

        expect(await deliverShare("sheet", image(), TEXT)).toBe("shared")
        expect(share.mock.calls[0][0]).toMatchObject({text: TEXT})
        expect(share.mock.calls[0][0].files).toHaveLength(1)
    })

    it("stops when the player closes the sheet, instead of delivering behind their back", async () => {
        const click = downloads()
        browser({
            share: vi.fn().mockRejectedValue(new DOMException("cancelled", "AbortError")),
            canShare: () => true,
        })

        expect(await deliverShare("sheet", image(), TEXT)).toBe("cancelled")
        expect(click).not.toHaveBeenCalled()
    })

    // The sheet is the only button a phone is shown, so a sheet that fell over
    // still has to leave the player holding the picture.
    it("downloads the file when the sheet itself fails", async () => {
        const click = downloads()
        browser({share: vi.fn().mockRejectedValue(new Error("the sheet fell over")), canShare: () => true})

        expect(await deliverShare("sheet", image(), TEXT)).toBe("downloaded")
        expect(click).toHaveBeenCalledTimes(1)
    })

    // The link is drawn into the image for this reason. Handed a clipboard
    // carrying the picture and the sentence, a chat window pastes the sentence.
    it("copies the picture alone, with no text for a paste target to prefer", async () => {
        const write = clipboard()
        browser({clipboard: {write}})

        expect(await deliverShare("copy", image(), TEXT)).toBe("copied")
        expect(write.mock.calls[0][0][0].types).toEqual(["image/png"])
    })

    // The download is its own button an inch away. A copy that turns into a
    // file in Downloads is the surprise the two buttons exist to stop.
    it("says a refused copy failed rather than quietly downloading instead", async () => {
        const click = downloads()
        clipboard()
        browser({clipboard: {write: vi.fn().mockRejectedValue(new Error("denied"))}})

        await expect(deliverShare("copy", image(), TEXT)).rejects.toThrow()
        expect(click).not.toHaveBeenCalled()
    })

    it("downloads under the file's own name", async () => {
        const click = downloads()
        browser({})

        expect(await deliverShare("download", image(), TEXT)).toBe("downloaded")
        expect(click).toHaveBeenCalledTimes(1)
    })
})
