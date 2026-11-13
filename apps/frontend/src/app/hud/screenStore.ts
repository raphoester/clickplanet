import {create} from "zustand"

export type MenuTab = "board" | "missions" | "share" | "invite" | "settings"

export type SheetName = "board" | "missions" | "chat" | "you" | "share" | "invite" | "settings" | "season" | "battle" | "clicks"

export type PopoverName = "season" | "battle" | "clicks"

export const MENU_OPEN_KEY = "clickplanet-menu-open"

export const CHAT_OPEN_KEY = "clickplanet-chat-open"

type ScreenStore = {
    sheet?: SheetName
    menuTab: MenuTab
    popovers: Readonly<Record<PopoverName, boolean>>
    // The desktop menu's panel and the chat as the player last left them; unset until the player opens or folds one.
    menuOpen?: boolean
    chatOpen?: boolean
    openSheet: (sheet: SheetName) => void
    toggleSheet: (sheet: SheetName) => void
    closeSheet: () => void
    setMenuTab: (menuTab: MenuTab) => void
    setMenuOpen: (open: boolean) => void
    setChatOpen: (open: boolean) => void
    togglePopover: (popover: PopoverName) => void
    closePopover: (popover: PopoverName) => void
    closeAll: () => void
}

const NO_POPOVER = {season: false, battle: false, clicks: false}

export function storedOpen(key: string, storage: () => Pick<Storage, "getItem"> = () => window.localStorage): boolean | undefined {
    try {
        const stored = storage().getItem(key)
        return stored === null ? undefined : stored !== "false"
    } catch {
        return undefined
    }
}

function keepOpen(key: string, open: boolean | undefined) {
    if (open === undefined) return
    try {
        window.localStorage.setItem(key, String(open))
    } catch {
        // A browser that keeps nothing keeps the choice for this visit only.
    }
}

export const useScreen = create<ScreenStore>()((set) => ({
    sheet: undefined,
    menuTab: "board",
    popovers: NO_POPOVER,
    menuOpen: storedOpen(MENU_OPEN_KEY),
    chatOpen: storedOpen(CHAT_OPEN_KEY),
    openSheet: (sheet) => set({sheet}),
    toggleSheet: (sheet) => set((state) => ({sheet: state.sheet === sheet ? undefined : sheet})),
    closeSheet: () => set({sheet: undefined}),
    setMenuTab: (menuTab) => set({menuTab}),
    setMenuOpen: (menuOpen) => set({menuOpen}),
    setChatOpen: (chatOpen) => set({chatOpen}),
    togglePopover: (popover) => set(({popovers}) => ({popovers: {...popovers, [popover]: !popovers[popover]}})),
    closePopover: (popover) => set(({popovers}) => ({popovers: {...popovers, [popover]: false}})),
    closeAll: () => set({sheet: undefined, popovers: NO_POPOVER}),
}))

useScreen.subscribe(({menuOpen, chatOpen}, was) => {
    if (menuOpen !== was.menuOpen) keepOpen(MENU_OPEN_KEY, menuOpen)
    if (chatOpen !== was.chatOpen) keepOpen(CHAT_OPEN_KEY, chatOpen)
})
