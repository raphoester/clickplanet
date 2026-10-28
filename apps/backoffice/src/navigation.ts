import {createContext, useContext} from "react"
import {Player} from "./gen/grpc/planet/v1/admin_pb.ts"
import {Caller} from "./players/caller.ts"

export type PageId =
    | "home" | "notes" | "reports" | "players" | "reassign" | "paint" | "replay" | "clips" | "titles" | "rules"
    | "antibotRules" | "schedule" | "seasons" | "timeline"

export type MenuGroup = {label: string, pages: {page: PageId, label: string}[]}

// Every page, as the burger menu lists them. A page's label is its title too.
export const MENU: MenuGroup[] = [
    {label: "Overview", pages: [{page: "home", label: "Home"}]},
    {label: "Feedback", pages: [{page: "notes", label: "Ideas and bugs"}]},
    {label: "Moderation", pages: [{page: "reports", label: "Reports"}, {page: "players", label: "Players"}]},
    {
        label: "Map",
        pages: [
            {page: "reassign", label: "Reassign a country"},
            {page: "paint", label: "Paint random tiles"},
            {page: "replay", label: "Download a replay"},
        ],
    },
    {label: "Clips", pages: [{page: "clips", label: "Clip stats"}]},
    {
        label: "Ops",
        pages: [
            {page: "timeline", label: "Timeline"},
            {page: "titles", label: "Reconcile titles"},
            {page: "rules", label: "Game rules"},
            {page: "antibotRules", label: "Anti-bot rules"},
            {page: "schedule", label: "Rule schedule"},
            {page: "seasons", label: "Seasons"},
        ],
    },
]

const PAGES: PageId[] = MENU.flatMap(group => group.pages.map(item => item.page))

export const FIRST_PAGE: PageId = "home"

export function titleOf(page: PageId): string {
    return MENU.flatMap(group => group.pages).find(item => item.page === page)?.label ?? page
}

// A page and what is open on it: a report's id, a player's account id or address, or "" for its list.
export type Route = {page: PageId, item: string}

// The route an address names (#/reports, #/players/203.0.113.7), and the first page for an address that names none.
export function routeOf(hash: string): Route {
    const [named, ...rest] = hash.replace(/^#\/?/, "").split("/")
    const page = PAGES.find(each => each === named)
    if (page === undefined) return {page: FIRST_PAGE, item: ""}
    return {page, item: decodeURIComponent(rest.join("/"))}
}

export function hashOf({page, item}: Route): string {
    return item === "" ? `#/${page}` : `#/${page}/${encodeURIComponent(item)}`
}

// A player to open on the Players page: who, and what the page that opens it already knows of it.
export type Opened = {caller: Caller, name?: string, row?: Player}

// Opens a player on the Players page from another page, as a report does. Unset outside App, as in a test of one page.
export const OpenPlayer = createContext<((opened: Opened) => void) | undefined>(undefined)

export function useOpenPlayer(): ((opened: Opened) => void) | undefined {
    return useContext(OpenPlayer)
}
