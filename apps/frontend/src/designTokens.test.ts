import {readdirSync, readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"

const SRC = fileURLToPath(new URL(".", import.meta.url))
const APP = fileURLToPath(new URL("..", import.meta.url))

const COLOR_PROPERTY = /^(?:color|background(?:-color|-image)?|border(?:-[a-z]+)*|outline(?:-color)?|box-shadow|text-shadow|fill|stroke|caret-color|accent-color|scrollbar-color|text-decoration(?:-color)?|column-rule(?:-color)?|--[\w-]+)$/
const HEX = /#[0-9a-f]{3,8}\b/i
const COLOR_FUNCTION = /\b(?:rgba?|hsla?)\(/i
const NAMED_COLOR = /\b(?:white|black|red|green|blue|yellow|orange|gray|grey|purple|pink|silver|gold|navy|teal)\b/i
const FONT_TOKEN = /var\(--font-(?:display|text)\)/

const TS_HEX = /["'`]#[0-9a-f]{6}(?:[0-9a-f]{2})?\b/i
const FONT_NAME = /Luckiest|Rubik|Oswald|Segoe|Tahoma|Helvetica|Verdana|Inter\b/

const BRANDS = new Set(["app/account/ProviderButton.tsx"])
const FONT_FACES = new Set(["app/titleFont.ts"])
const DOM_IN_VIEWER = new Set(["app/viewer/bonusPointer.ts", "app/viewer/questionMark.ts", "app/viewer/blastMark.ts"])

type Declaration = {property: string, value: string}

function read(path: string): string {
    return readFileSync(path, "utf8")
}

function filesUnder(dir: string, extensions: RegExp): string[] {
    return readdirSync(dir, {recursive: true, encoding: "utf8"})
        .filter((file) => extensions.test(file) && !file.startsWith("gen/"))
}

function declarationsOf(css: string): Declaration[] {
    const stripped = css.replace(/\/\*[\s\S]*?\*\//g, "")
    return [...stripped.matchAll(/\{([^{}]*)\}/g)].flatMap((block) => block[1].split(";").flatMap((part) => {
        const colon = part.indexOf(":")
        if (colon < 0) return []
        return [{property: part.slice(0, colon).trim().toLowerCase(), value: part.slice(colon + 1).trim()}]
    }))
}

function offences(css: string): string[] {
    return declarationsOf(css).flatMap(({property, value}) => {
        const found: string[] = []
        if (HEX.test(value)) found.push("a hex color")
        if (COLOR_FUNCTION.test(value) && !value.includes("var(--author-hue")) found.push("a color function")
        if (COLOR_PROPERTY.test(property) && NAMED_COLOR.test(value.replace(/--[\w-]+/g, ""))) found.push("a named color")
        if (property === "font-family" && !FONT_TOKEN.test(value) && value !== "inherit") found.push("a font name")
        if (property === "font" && !FONT_TOKEN.test(value) && value !== "inherit") found.push("a font name")
        return found.map((what) => `${property}: ${value} (${what})`)
    })
}

function stylesOf(html: string): string {
    return [...html.matchAll(/<style>([\s\S]*?)<\/style>/g)].map((match) => match[1]).join("\n")
}

describe("the design tokens", () => {
    it("are the only place a stylesheet of the game names a color or a font", () => {
        const found = filesUnder(SRC, /\.css$/)
            .filter((file) => file !== "tokens.css")
            .flatMap((file) => offences(read(`${SRC}${file}`)).map((offence) => `${file}: ${offence}`))

        expect(found).toEqual([])
    })

    for (const page of ["index.html", "privacy.html", "terms.html"]) {
        it(`are the only place ${page} names a color or a font`, () => {
            const html = read(`${APP}${page}`)

            expect(html).toContain(`<link rel="stylesheet" href="/src/tokens.css">`)
            expect(offences(stylesOf(html))).toEqual([])
        })
    }

    it("are the only colors a component draws, other companies' brands aside", () => {
        const found = filesUnder(`${SRC}app`, /\.tsx?$/)
            .map((file) => `app/${file}`)
            .filter((file) => !file.includes(".test.") && !BRANDS.has(file))
            .filter((file) => !file.startsWith("app/viewer/") || DOM_IN_VIEWER.has(file))
            .filter((file) => TS_HEX.test(read(`${SRC}${file}`)))

        expect(found).toEqual([])
    })

    it("name the two faces in one place outside the stylesheets", () => {
        const found = filesUnder(`${SRC}app`, /\.tsx?$/)
            .map((file) => `app/${file}`)
            .filter((file) => !file.includes(".test.") && !FONT_FACES.has(file))
            .filter((file) => FONT_NAME.test(read(`${SRC}${file}`)))

        expect(found).toEqual([])
    })
})
