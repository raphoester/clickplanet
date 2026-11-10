import {create} from "zustand"

export type MenuTab = "board" | "missions" | "share" | "invite" | "settings"

export type SheetName = "board" | "missions" | "chat" | "you" | "share" | "invite" | "settings" | "season" | "battle" | "clicks"

export type PopoverName = "season" | "battle" | "clicks"

type ScreenStore = {
    sheet?: SheetName
    menuTab: MenuTab
    popovers: Readonly<Record<PopoverName, boolean>>
    openSheet: (sheet: SheetName) => void
    toggleSheet: (sheet: SheetName) => void
    closeSheet: () => void
    setMenuTab: (menuTab: MenuTab) => void
    togglePopover: (popover: PopoverName) => void
    closePopover: (popover: PopoverName) => void
    closeAll: () => void
}

const NO_POPOVER = {season: false, battle: false, clicks: false}

export const useScreen = create<ScreenStore>()((set) => ({
    sheet: undefined,
    menuTab: "board",
    popovers: NO_POPOVER,
    openSheet: (sheet) => set({sheet}),
    toggleSheet: (sheet) => set((state) => ({sheet: state.sheet === sheet ? undefined : sheet})),
    closeSheet: () => set({sheet: undefined}),
    setMenuTab: (menuTab) => set({menuTab}),
    togglePopover: (popover) => set(({popovers}) => ({popovers: {...popovers, [popover]: !popovers[popover]}})),
    closePopover: (popover) => set(({popovers}) => ({popovers: {...popovers, [popover]: false}})),
    closeAll: () => set({sheet: undefined, popovers: NO_POPOVER}),
}))
