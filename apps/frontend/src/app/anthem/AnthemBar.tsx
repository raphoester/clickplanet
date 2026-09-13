import {useEffect, useRef} from "react";
import {Countries} from "../../domain/countries.ts";
import {isAnthemAudible, SoundSettings} from "../../domain/soundSettings.ts";
import CountryFlag from "../components/CountryFlag.tsx";
import {PauseIcon, PlayIcon} from "../components/icons.tsx";
import {Anthem} from "./useAnthem.ts";
import "./AnthemBar.css"

const PROGRESS_MS = 250

export type AnthemBarProps = {
    anthem: Anthem
    settings: SoundSettings
    onChange: (settings: SoundSettings) => void
}

/**
 * The music player at the bottom of the screen: whose anthem is playing, and
 * the play button and volume for it. Both write the sound settings, so the
 * player and the settings panel never disagree.
 */
export default function AnthemBar({anthem, settings, onChange}: AnthemBarProps) {
    const progress = useRef<HTMLDivElement>(null)
    const recorded = anthem.title !== undefined
    const playing = isAnthemAudible(settings) && recorded

    // The bar is moved by a CSS variable, not by state: four renders a second
    // for a sliver of colour is not worth it beside the globe.
    useEffect(() => {
        if (!playing) return
        const timer = setInterval(() => {
            const at = anthem.player.position()
            progress.current?.style.setProperty(
                "--anthem-progress", at ? String(at.current / at.duration) : "0")
        }, PROGRESS_MS)
        return () => clearInterval(timer)
    }, [playing, anthem.player])

    const code = anthem.code
    if (!code) return null

    const country = Countries.get(code)?.name ?? code.toUpperCase()

    // Pressing play is asking to hear it, so it also lifts the master switch
    // if that was what kept it quiet. Pausing touches only the anthem.
    const toggle = () => onChange(playing
        ? {...settings, anthem: {...settings.anthem, on: false}}
        : {
            ...settings,
            enabled: true,
            anthem: {on: true, volume: settings.anthem.volume > 0 ? settings.anthem.volume : 0.3},
        })

    const setVolume = (volume: number) =>
        onChange({...settings, anthem: {on: volume > 0 || settings.anthem.on, volume}})

    return <section className={playing ? "anthem-bar anthem-bar--playing" : "anthem-bar"}
                    aria-label="National anthem">
        <button type="button"
                className="anthem-bar-play"
                aria-label={playing ? "Pause the anthem" : "Play the anthem"}
                disabled={!recorded}
                onClick={toggle}>
            {playing ? <PauseIcon/> : <PlayIcon/>}
        </button>

        <div className="anthem-bar-text">
            <div className="anthem-bar-title">
                <CountryFlag code={code}/>
                <span className="anthem-bar-name">{anthem.title ?? country}</span>
            </div>
            <div className="anthem-bar-subtitle">{subtitle(anthem, playing, country)}</div>
        </div>

        <input type="range"
               className="anthem-bar-volume"
               aria-label="Anthem volume"
               min={0} max={1} step={0.05}
               value={settings.enabled && settings.anthem.on ? settings.anthem.volume : 0}
               disabled={!recorded}
               onChange={(event) => setVolume(Number(event.target.value))}/>

        <div ref={progress} className="anthem-bar-progress" aria-hidden="true"/>
    </section>
}

function subtitle(anthem: Anthem, playing: boolean, country: string): string {
    if (anthem.title === undefined) return "No recording of this anthem"
    if (playing && !anthem.unlocked) return "Click anywhere to start the music"
    return `${country} · US Navy Band`
}
