import {describe, expect, it} from "vitest"
import {finaleLinks} from "./seasonCalendar.ts"

const season = {
    number: 0,
    finaleStartsAt: Date.UTC(2026, 9, 31, 21),
    endsAt: Date.UTC(2026, 9, 31, 23),
    finaleFile: "https://api.clickplanet.lol/seasons/0/finale.ics",
}
const playUrl = "https://clickplanet.lol/play"

const linkTo = (name: string, of = season) => finaleLinks(of, playUrl).find(link => link.name === name)!

describe("finaleLinks", () => {
    it("offers Google, Apple and Outlook, in that order", () => {
        expect(finaleLinks(season, playUrl).map(link => link.name)).toEqual(["Google Calendar", "Apple Calendar", "Outlook"])
    })

    it("opens Google Calendar on a new event, in UTC, with a link to the game", () => {
        const link = linkTo("Google Calendar")

        expect(link.web).toBe(true)
        expect(link.url).toBe("https://calendar.google.com/calendar/render?action=TEMPLATE" +
            "&text=ClickPlanet%20Season%200%3A%20Final%20Battle" +
            "&dates=20261031T210000Z%2F20261031T230000Z" +
            "&details=https%3A%2F%2Fclickplanet.lol%2Fplay")
    })

    it("opens Outlook on a new event, in UTC, with a link to the game", () => {
        const link = linkTo("Outlook")

        expect(link.web).toBe(true)
        expect(link.url).toBe("https://outlook.live.com/calendar/0/deeplink/compose" +
            "?path=%2Fcalendar%2Faction%2Fcompose&rru=addevent" +
            "&subject=ClickPlanet%20Season%200%3A%20Final%20Battle" +
            "&startdt=2026-10-31T21%3A00%3A00Z&enddt=2026-10-31T23%3A00%3A00Z" +
            "&body=https%3A%2F%2Fclickplanet.lol%2Fplay")
    })

    it("hands Apple Calendar the server's file", () => {
        const link = linkTo("Apple Calendar")

        expect(link.web).toBe(false)
        expect(link.url).toBe("https://api.clickplanet.lol/seasons/0/finale.ics")
    })

    it("leaves Apple Calendar out when no file is served", () => {
        const links = finaleLinks({...season, finaleFile: undefined}, playUrl)

        expect(links.map(link => link.name)).toEqual(["Google Calendar", "Outlook"])
    })

    it("takes its dates and its number from the season", () => {
        const later = {...season, number: 2, finaleStartsAt: Date.UTC(2027, 0, 31, 20, 30), endsAt: Date.UTC(2027, 0, 31, 23)}
        const google = new URL(linkTo("Google Calendar", later).url).searchParams
        const outlook = new URL(linkTo("Outlook", later).url).searchParams

        expect(google.get("text")).toBe("ClickPlanet Season 2: Final Battle")
        expect(google.get("dates")).toBe("20270131T203000Z/20270131T230000Z")
        expect(outlook.get("subject")).toBe("ClickPlanet Season 2: Final Battle")
        expect(outlook.get("startdt")).toBe("2027-01-31T20:30:00Z")
        expect(outlook.get("enddt")).toBe("2027-01-31T23:00:00Z")
    })
})
