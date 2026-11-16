import {createContext} from "react"

// The row inside the dock, above the clicks, where a tray opens: the dock grows with it rather than cover anything.
export const DockTray = createContext<HTMLElement | null>(null)
