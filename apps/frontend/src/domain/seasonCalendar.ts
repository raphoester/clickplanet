import {Season} from "../backends/season.ts"

export type CalendarLink = {
    name: string
    url: string
    web: boolean
}

export function finaleLinks(season: Season, playUrl: string): CalendarLink[] {
    const title = `ClickPlanet Season ${season.number}: Final Battle`

    const google = query({
        action: "TEMPLATE",
        text: title,
        dates: `${compact(season.finaleStartsAt)}/${compact(season.endsAt)}`,
        details: playUrl,
    })
    const outlook = query({
        path: "/calendar/action/compose",
        rru: "addevent",
        subject: title,
        startdt: iso(season.finaleStartsAt),
        enddt: iso(season.endsAt),
        body: playUrl,
    })

    return [
        {name: "Google Calendar", url: `https://calendar.google.com/calendar/render?${google}`, web: true},
        ...(season.finaleFile ? [{name: "Apple Calendar", url: season.finaleFile, web: false}] : []),
        {name: "Outlook", url: `https://outlook.live.com/calendar/0/deeplink/compose?${outlook}`, web: true},
    ]
}

function query(params: Record<string, string>): string {
    return Object.entries(params).map(([key, value]) => `${key}=${encodeURIComponent(value)}`).join("&")
}

function iso(ms: number): string {
    return new Date(ms).toISOString().replace(/\.\d{3}/, "")
}

function compact(ms: number): string {
    return iso(ms).replace(/[-:]/g, "")
}
