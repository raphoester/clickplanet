import {useEffect, useRef, useState} from "react";
import {LeaderboardEntry} from "../../domain/leaderboard.ts";
import {AnthemPick, followLeader, NO_PICK} from "../../domain/anthemLeader.ts";
import {isAnthemAudible, SoundSettings} from "../../domain/soundSettings.ts";
import {ANTHEMS} from "./anthemsAsset.ts";
import {AnthemPlayer, createAnthemPlayer} from "./anthemPlayer.ts";

const HOLD_CHECK_MS = 1000

export type Anthem = {
    code: string | undefined
    title: string | undefined
    unlocked: boolean
    player: AnthemPlayer
}

export function useAnthem(leaderboard: readonly LeaderboardEntry[], settings: SoundSettings): Anthem {
    const playerRef = useRef<AnthemPlayer | null>(null)
    if (!playerRef.current) playerRef.current = createAnthemPlayer()
    const player = playerRef.current

    const [pick, setPick] = useState<AnthemPick>(NO_PICK)
    const [unlocked, setUnlocked] = useState(false)
    const [hidden, setHidden] = useState(() => typeof document !== "undefined" && document.hidden)

    const leader = leaderboard[0]?.country.code
    const leaderRef = useRef(leader)
    leaderRef.current = leader

    useEffect(() => {
        setPick((standing) => followLeader(standing, leader, Date.now()))
    }, [leader])

    const waiting = pick.challenger !== undefined
    useEffect(() => {
        if (!waiting) return
        const timer = setInterval(
            () => setPick((standing) => followLeader(standing, leaderRef.current, Date.now())),
            HOLD_CHECK_MS)
        return () => clearInterval(timer)
    }, [waiting])

    const recording = pick.playing ? ANTHEMS[pick.playing] : undefined
    const url = recording?.url
    useEffect(() => player.setTrack(url), [player, url])

    const audible = isAnthemAudible(settings) && !hidden
    useEffect(() => player.setAudible(audible), [player, audible])
    useEffect(() => player.setVolume(settings.anthem.volume), [player, settings.anthem.volume])

    useEffect(() => {
        const onVisibility = () => setHidden(document.hidden)
        document.addEventListener("visibilitychange", onVisibility)
        return () => document.removeEventListener("visibilitychange", onVisibility)
    }, [])

    useEffect(() => {
        const unlock = () => {
            player.unlock()
            setUnlocked(true)
        }
        const options = {capture: true}
        window.addEventListener("pointerdown", unlock, options)
        window.addEventListener("keydown", unlock, options)
        return () => {
            window.removeEventListener("pointerdown", unlock, options)
            window.removeEventListener("keydown", unlock, options)
        }
    }, [player])

    useEffect(() => () => player.dispose(), [player])

    return {code: pick.playing, title: recording?.title, unlocked, player}
}
