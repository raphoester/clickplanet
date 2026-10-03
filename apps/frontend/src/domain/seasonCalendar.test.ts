import {describe, expect, it} from "vitest"
import {finaleCalendar} from "./seasonCalendar.ts"

const season = {number: 0, finaleStartsAt: Date.UTC(2026, 9, 31, 21), endsAt: Date.UTC(2026, 9, 31, 23)}
const now = Date.UTC(2026, 9, 3, 10, 15, 30, 250)

describe("finaleCalendar", () => {
    it("is one event, the finale, in UTC, with a link to the game", () => {
        const file = finaleCalendar(season, "https://clickplanet.lol/play", now)

        expect(file.name).toBe("clickplanet-season-0-finale.ics")
        expect(file.text).toBe([
            "BEGIN:VCALENDAR",
            "VERSION:2.0",
            "PRODID:-//ClickPlanet//Seasons//EN",
            "CALSCALE:GREGORIAN",
            "METHOD:PUBLISH",
            "BEGIN:VEVENT",
            "UID:season-0-finale@clickplanet.lol",
            "DTSTAMP:20261003T101530Z",
            "DTSTART:20261031T210000Z",
            "DTEND:20261031T230000Z",
            "SUMMARY:ClickPlanet Season 0: Final Assault",
            "DESCRIPTION:https://clickplanet.lol/play",
            "URL:https://clickplanet.lol/play",
            "END:VEVENT",
            "END:VCALENDAR",
            "",
        ].join("\r\n"))
    })

    it("takes its dates and its number from the season", () => {
        const file = finaleCalendar(
            {number: 2, finaleStartsAt: Date.UTC(2027, 0, 31, 20, 30), endsAt: Date.UTC(2027, 0, 31, 23)},
            "https://clickplanet.lol/play",
            now)

        expect(file.name).toBe("clickplanet-season-2-finale.ics")
        expect(file.text).toContain("\r\nUID:season-2-finale@clickplanet.lol\r\n")
        expect(file.text).toContain("\r\nDTSTART:20270131T203000Z\r\nDTEND:20270131T230000Z\r\n")
        expect(file.text).toContain("\r\nSUMMARY:ClickPlanet Season 2: Final Assault\r\n")
    })

    it("escapes what a text value may not hold", () => {
        const file = finaleCalendar(season, "https://example.com/play?a=1,b;c\\d", now)

        expect(file.text).toContain("\r\nDESCRIPTION:https://example.com/play?a=1\\,b\\;c\\\\d\r\n")
    })

    it("keeps every line within 75 octets", () => {
        const file = finaleCalendar(season, "https://clickplanet.lol/play", now)

        for (const line of file.text.split("\r\n")) expect(new TextEncoder().encode(line).length).toBeLessThanOrEqual(75)
    })
})
