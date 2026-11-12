import {useEffect, useMemo, useState} from "react"
import {RecentHistorySource} from "../../backends/recentHistory.ts"
import {dayOf} from "../../domain/video/dots.ts"
import {LOSS_MOMENT_SHARE, movedShare, storyOf, VideoStory} from "../../domain/video/story.ts"
import {encoderConfig} from "../../video/job.ts"
import {ListenForClicks} from "../clicks/acceptedClicks.ts"
import {useAnnouncements} from "../announcements/announcementsStore.ts"
import {readHistory, useVideo} from "./videoStore.ts"

const LOSS_KEY = "clickplanet-video-loss"

// Long enough that the map's first reads and draws are done.
const IDLE_TIMEOUT_MS = 10_000

export type VideoOffer = {
    // The tile and the moments' buttons are there.
    offered: boolean
    dot: boolean
    story?: VideoStory
}

export type VideoOfferProps = {
    source: RecentHistorySource
    ready: boolean
    flag: string
    // No arena plays, whose map is not the world's the 24 hours are of, and the page is not CrazyGames', whose
    // players the address would lead away.
    allowed: boolean
    listenForClicks: ListenForClicks
}

let encodes: Promise<boolean> | undefined

export function videoEncodes(): Promise<boolean> {
    encodes ??= encoderConfig().then((config) => config !== undefined, () => false)
    return encodes
}

function whenIdle(run: () => void): () => void {
    if (typeof window.requestIdleCallback === "function") {
        const id = window.requestIdleCallback(run, {timeout: IDLE_TIMEOUT_MS})
        return () => window.cancelIdleCallback(id)
    }
    const id = window.setTimeout(run, IDLE_TIMEOUT_MS / 3)
    return () => window.clearTimeout(id)
}

function lossShownOn(): string | null {
    try {
        return window.localStorage.getItem(LOSS_KEY)
    } catch {
        return null
    }
}

function keepLossShown(day: string) {
    try {
        window.localStorage.setItem(LOSS_KEY, day)
    } catch {
        // storage unavailable: the moment may come again on a reload
    }
}

// Who clicked today can make a video of its flag's day, where the browser encodes H.264 and no arena plays. The 24 hours
// are read once the map is up and the page idle, never on a render.
export function useVideoOffer({source, ready, flag, allowed, listenForClicks}: VideoOfferProps): VideoOffer {
    const [encoding, setEncoding] = useState(false)
    useEffect(() => {
        let live = true
        void videoEncodes().then((yes) => {
            if (live) setEncoding(yes)
        })
        return () => {
            live = false
        }
    }, [])

    const clickedOn = useVideo((video) => video.clickedOn)
    useEffect(() => listenForClicks(() => clickedOn(dayOf(Date.now()))), [listenForClicks, clickedOn])
    const clicked = useVideo((video) => video.clicked)
    const offered = encoding && allowed && clicked === dayOf(Date.now())

    const read = useVideo((video) => video.read)
    useEffect(() => {
        if (!ready || !offered || read !== undefined) return
        return whenIdle(() => void readHistory(source).catch((e) => console.error("could not read the map's last 24 hours", e)))
    }, [ready, offered, read, source])

    const story = useMemo(() => read && storyOf(read.history, read.ground, flag), [read, flag])

    const light = useVideo((video) => video.light)
    useEffect(() => {
        if (story) light("visit", dayOf(Date.now()), movedShare(story))
    }, [story, light])

    const announce = useAnnouncements((announcements) => announcements.announce)
    useEffect(() => {
        if (!offered || !story || story.kind !== "loss" || movedShare(story) < LOSS_MOMENT_SHARE) return
        const today = dayOf(Date.now())
        if (lossShownOn() === today) return
        keepLossShown(today)
        announce({kind: "video", story})
    }, [offered, story, announce])

    const dot = useVideo((video) => video.dots.on)
    return {offered, dot: offered && dot, story}
}
