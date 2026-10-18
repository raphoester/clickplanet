import {createContext, useContext} from "react"
import {Player} from "./gen/grpc/planet/v1/admin_pb.ts"
import {Caller} from "./players/caller.ts"

export type PageId = "notes" | "reports" | "players" | "reassign" | "paint" | "replay" | "titles" | "names"

export type MenuGroup = {label: string, pages: {page: PageId, label: string}[]}

// Every page, as the burger menu lists them. A page's label is its title too.
export const MENU: MenuGroup[] = [
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
    {label: "Ops", pages: [{page: "titles", label: "Reconcile titles"}, {page: "names", label: "Name accounts"}]},
]

export function titleOf(page: PageId): string {
    return MENU.flatMap(group => group.pages).find(item => item.page === page)?.label ?? page
}

// A player to open on the Players page: who, and what the page that opens it already knows of it.
export type Opened = {caller: Caller, name?: string, row?: Player}

// Opens a player on the Players page from another page, as a report does. Unset outside App, as in a test of one page.
export const OpenPlayer = createContext<((opened: Opened) => void) | undefined>(undefined)

export function useOpenPlayer(): ((opened: Opened) => void) | undefined {
    return useContext(OpenPlayer)
}
