// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {deliveriesOffered, deliverLink, deliverShare, linkDeliveryOffered} from "./deliverShare.ts"

const image = () => new File([new Uint8Array([1, 2, 3])], "clickplanet-fr.png", {type: "image/png"})
const TEXT = "France is #2 on ClickPlanet. https://clickplanet.lol/?f=fr"
const LINK = "https://clickplanet.lol/?f=fr"
const HEADLINE = "France is #2 on ClickPlanet."

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

function phone(coarse: boolean) {
    vi.stubGlobal("matchMedia", (query: string) =>
        ({matches: coarse && query === "(pointer: coarse)"}))
}

function clipboard(write = vi.fn().mockResolvedValue(undefined)) {
    vi.stubGlobal("ClipboardItem", class {
        types: string[]
        constructor(public items: Record<string, Blob>) {
            this.types = Object.keys(items)
        }
    })
    return write
}

function downloads() {
    URL.createObjectURL = vi.fn().mockReturnValue("blob:fake")
    URL.revokeObjectURL = vi.fn()
    return vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {})
}

function selectionCopy(copies: boolean) {
    const execCommand = vi.fn().mockReturnValue(copies)
    Object.defineProperty(document, "execCommand", {value: execCommand, configurable: true, writable: true})
    return execCommand
}

afterEach(() => {
    browser({})
    Reflect.deleteProperty(document, "execCommand")
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

    it("keeps a desktop off the share sheet, whatever it claims it can share", () => {
        phone(false)
        browser({share: vi.fn(), canShare: () => true, clipboard: {write: vi.fn()}})
        clipboard()

        expect(deliveriesOffered()).toEqual(["copy", "download"])
    })

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

    it("downloads the file when the sheet itself fails", async () => {
        const click = downloads()
        browser({share: vi.fn().mockRejectedValue(new Error("the sheet fell over")), canShare: () => true})

        expect(await deliverShare("sheet", image(), TEXT)).toBe("downloaded")
        expect(click).toHaveBeenCalledTimes(1)
    })

    it("copies the picture alone, with no text for a paste target to prefer", async () => {
        const write = clipboard()
        browser({clipboard: {write}})

        expect(await deliverShare("copy", image(), TEXT)).toBe("copied")
        expect(write.mock.calls[0][0][0].types).toEqual(["image/png"])
    })

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

describe("what a browser is offered for a link", () => {
    it("gives a phone its share sheet", () => {
        phone(true)
        browser({share: vi.fn()})

        expect(linkDeliveryOffered()).toBe("sheet")
    })

    it("copies on a desktop, whatever it claims it can share", () => {
        phone(false)
        browser({share: vi.fn()})

        expect(linkDeliveryOffered()).toBe("copy")
    })

    it("copies on a phone with no share sheet", () => {
        phone(true)
        browser({})

        expect(linkDeliveryOffered()).toBe("copy")
    })
})

describe("delivering the link", () => {
    it("hands the link and the sentence to the share sheet, apart", async () => {
        const share = vi.fn().mockResolvedValue(undefined)
        browser({share})

        expect(await deliverLink("sheet", LINK, HEADLINE)).toBe("shared")
        expect(share).toHaveBeenCalledWith({title: "ClickPlanet", text: HEADLINE, url: LINK})
    })

    it("stops when the player closes the sheet, instead of copying behind their back", async () => {
        const writeText = vi.fn().mockResolvedValue(undefined)
        browser({share: vi.fn().mockRejectedValue(new DOMException("cancelled", "AbortError")), clipboard: {writeText}})

        expect(await deliverLink("sheet", LINK, HEADLINE)).toBe("cancelled")
        expect(writeText).not.toHaveBeenCalled()
    })

    it("copies the link when the sheet itself fails", async () => {
        const writeText = vi.fn().mockResolvedValue(undefined)
        browser({share: vi.fn().mockRejectedValue(new Error("the sheet fell over")), clipboard: {writeText}})

        expect(await deliverLink("sheet", LINK, HEADLINE)).toBe("copied")
        expect(writeText).toHaveBeenCalledWith(LINK)
    })

    it("copies the link alone, with no sentence around it", async () => {
        const writeText = vi.fn().mockResolvedValue(undefined)
        browser({clipboard: {writeText}})

        expect(await deliverLink("copy", LINK, HEADLINE)).toBe("copied")
        expect(writeText).toHaveBeenCalledWith(LINK)
    })

    it("copies a selection where the clipboard is refused, and leaves nothing on the page", async () => {
        const execCommand = selectionCopy(true)
        browser({clipboard: {writeText: vi.fn().mockRejectedValue(new Error("denied"))}})

        expect(await deliverLink("copy", LINK, HEADLINE)).toBe("copied")
        expect(execCommand).toHaveBeenCalledWith("copy")
        expect(document.querySelector("textarea")).toBeNull()
    })

    it("copies a selection in a browser with no clipboard at all", async () => {
        selectionCopy(true)
        browser({})

        expect(await deliverLink("copy", LINK, HEADLINE)).toBe("copied")
    })

    it("says a copy that nothing would take failed", async () => {
        selectionCopy(false)
        browser({clipboard: {writeText: vi.fn().mockRejectedValue(new Error("denied"))}})

        await expect(deliverLink("copy", LINK, HEADLINE)).rejects.toThrow()
        expect(document.querySelector("textarea")).toBeNull()
    })
})
