import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"
import {firstColumn, wordsOf} from "./crazygames/sheet.mjs"

const SHEET = "https://docs.google.com/spreadsheets/d/1w8HNWjNdPO_Bd_ob8XmIuABG08qdwJwn/export?format=csv"
const FEWEST_WORDS = 1_000

const out = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "public", "crazygames", "words.txt")

const response = await fetch(SHEET)
const type = response.headers.get("content-type") ?? ""
if (!response.ok || !type.startsWith("text/csv")) {
    throw new Error(`CrazyGames' word list answered ${response.status} ${type}`)
}
const words = wordsOf(firstColumn(await response.text()))
if (words.length < FEWEST_WORDS) {
    throw new Error(`CrazyGames' word list has ${words.length} words, fewer than ${FEWEST_WORDS}`)
}

fs.mkdirSync(path.dirname(out), {recursive: true})
fs.writeFileSync(out, `${words.join("\n")}\n`)
console.log(`wrote ${words.length} words of CrazyGames' list to ${out}`)
