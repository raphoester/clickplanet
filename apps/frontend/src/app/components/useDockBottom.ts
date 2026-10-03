import {useBottomEdge} from "./useBottomEdge.ts";

export const DOCK_BOTTOM = "--click-budget-dock-bottom"

export function useDockBottom(dock: HTMLElement | null) {
    useBottomEdge(dock, DOCK_BOTTOM)
}
