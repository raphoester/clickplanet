import {PageContext} from "../../backends/support.ts"
import {GraphicsReport} from "../viewer/graphicsReport.ts"

export function buildOf(moduleUrl: string): string {
    try {
        return new URL(moduleUrl).pathname.split("/").pop() ?? ""
    } catch {
        return ""
    }
}

// The file the bundler put this module in: its name carries the build's hash.
export const BUILD = buildOf(import.meta.url)

export type Device = {
    innerWidth: number
    innerHeight: number
    devicePixelRatio?: number
    navigator: {
        hardwareConcurrency?: number
        // Chromium only.
        deviceMemory?: number
    }
}

export function pageContextOf(device: Device, graphics: GraphicsReport, connectionDown: boolean): PageContext {
    return {
        build: BUILD,
        viewportWidth: Math.round(device.innerWidth),
        viewportHeight: Math.round(device.innerHeight),
        pixelRatio: device.devicePixelRatio ?? 0,
        connectionDown,
        gpuVendor: graphics.vendor,
        gpuRenderer: graphics.renderer,
        webglVersion: graphics.version,
        maxTextureSize: graphics.maxTextureSize,
        contextLost: graphics.contextLost,
        cores: device.navigator.hardwareConcurrency ?? 0,
        memoryGb: device.navigator.deviceMemory ?? 0,
    }
}
