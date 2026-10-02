import {PointerEvent, RefObject, useLayoutEffect} from "react";
import {CHAT_SIZE_STORAGE_KEY, ChatSize, draggedSize, parseStoredSize, ResizeEdge} from "./chatSize.ts";

export const WANTED_WIDTH = "--chat-wanted-width"
export const WANTED_HEIGHT = "--chat-wanted-height"

export function useChatSize(panel: RefObject<HTMLElement | null>) {
    useLayoutEffect(() => {
        const stored = parseStoredSize(readStoredSize())
        if (stored) applySize(stored)
        return clearSize
    }, [])

    const startResize = (edge: ResizeEdge, event: PointerEvent<HTMLElement>) => {
        const element = panel.current
        if (!element || event.button !== 0) return
        event.preventDefault()

        const handle = event.currentTarget
        const {width, height} = element.getBoundingClientRect()
        const {clientX, clientY} = event
        handle.setPointerCapture(event.pointerId)

        let moved = false
        const move = (pointer: globalThis.PointerEvent) => {
            moved = true
            applySize(draggedSize(edge, {width, height}, pointer.clientX - clientX, pointer.clientY - clientY))
        }

        const end = () => {
            handle.removeEventListener("pointermove", move)
            handle.removeEventListener("lostpointercapture", end)
            if (!moved) return

            const shown = element.getBoundingClientRect()
            const size = {width: shown.width, height: shown.height}
            applySize(size)
            writeStoredSize(size)
        }

        handle.addEventListener("pointermove", move)
        handle.addEventListener("lostpointercapture", end)
    }

    const resetSize = () => {
        clearSize()
        clearStoredSize()
    }

    return {startResize, resetSize}
}

function applySize(size: ChatSize): void {
    const style = document.documentElement.style
    style.setProperty(WANTED_WIDTH, `${Math.round(size.width)}px`)
    style.setProperty(WANTED_HEIGHT, `${Math.round(size.height)}px`)
}

function clearSize(): void {
    const style = document.documentElement.style
    style.removeProperty(WANTED_WIDTH)
    style.removeProperty(WANTED_HEIGHT)
}

function readStoredSize(): string | null {
    try {
        return window.localStorage.getItem(CHAT_SIZE_STORAGE_KEY)
    } catch {
        return null
    }
}

function writeStoredSize(size: ChatSize): void {
    try {
        window.localStorage.setItem(CHAT_SIZE_STORAGE_KEY, JSON.stringify(size))
    } catch {
        // storage unavailable
    }
}

function clearStoredSize(): void {
    try {
        window.localStorage.removeItem(CHAT_SIZE_STORAGE_KEY)
    } catch {
        // storage unavailable
    }
}
