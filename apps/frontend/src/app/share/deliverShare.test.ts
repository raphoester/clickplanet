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
