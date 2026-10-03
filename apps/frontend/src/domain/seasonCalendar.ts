import {Season} from "../backends/season.ts"

export type CalendarFile = {
    name: string
    text: string
}

export function finaleCalendar(season: Season, playUrl: string, now: number): CalendarFile {
    const lines = [
        "BEGIN:VCALENDAR",
        "VERSION:2.0",
        "PRODID:-//ClickPlanet//Seasons//EN",
        "CALSCALE:GREGORIAN",
        "METHOD:PUBLISH",
        "BEGIN:VEVENT",
        `UID:season-${season.number}-finale@clickplanet.lol`,
        `DTSTAMP:${utc(now)}`,
        `DTSTART:${utc(season.finaleStartsAt)}`,
        `DTEND:${utc(season.endsAt)}`,
        `SUMMARY:${text(`ClickPlanet Season ${season.number}: Final Battle`)}`,
        `DESCRIPTION:${text(playUrl)}`,
        `URL:${playUrl}`,
        "END:VEVENT",
        "END:VCALENDAR",
    ]

    return {
        name: `clickplanet-season-${season.number}-finale.ics`,
        text: lines.map((line) => `${line}\r\n`).join(""),
    }
}

function utc(ms: number): string {
    return new Date(ms).toISOString().replace(/[-:]/g, "").replace(/\.\d{3}/, "")
}

function text(value: string): string {
    return value.replace(/[\\;,]/g, (c) => `\\${c}`).replace(/\n/g, "\\n")
}
