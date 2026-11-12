import {useEffect, useRef} from "react";
import {isAnthemAudible} from "../../domain/soundSettings.ts";
import CountryFlag from "../components/CountryFlag.tsx";
import {PauseIcon, PlayIcon} from "../components/icons.tsx";
import {useSoundSettings} from "../sound/soundSettingsStore.ts";
import {Anthem} from "./useAnthem.ts";
import {t, translator} from "../../i18n/page.ts";
import "./AnthemControls.css"

const PROGRESS_MS = 250

export type AnthemControlsProps = {
    anthem: Anthem
}

export default function AnthemControls({anthem}: AnthemControlsProps) {
    const {settings, setSettings} = useSoundSettings()
    const progress = useRef<HTMLDivElement>(null)
    const recorded = anthem.title !== undefined
    const playing = isAnthemAudible(settings) && recorded

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

    const country = translator().country(code)

    const toggle = () => setSettings(playing
        ? {...settings, anthem: {...settings.anthem, on: false}}
        : {
            ...settings,
            enabled: true,
            anthem: {on: true, volume: settings.anthem.volume > 0 ? settings.anthem.volume : 0.3},
        })

    const setVolume = (volume: number) => setSettings({
        ...settings,
        enabled: volume > 0 || settings.enabled,
        anthem: {on: volume > 0 || settings.anthem.on, volume},
    })

    return <section className={playing ? "anthem anthem--playing" : "anthem"} aria-label={t("anthem.section")}>
        <button type="button"
                className="icon-button anthem-play"
                aria-label={playing ? t("anthem.pause") : t("anthem.play")}
                disabled={!recorded}
                onClick={toggle}>
            {playing ? <PauseIcon/> : <PlayIcon/>}
        </button>

        <div className="anthem-text">
            <div className="anthem-title">
                <CountryFlag code={code}/>
                <span className="anthem-name">{anthem.title ?? country}</span>
            </div>
            <div className="anthem-subtitle">{subtitle(anthem, playing, country)}</div>
        </div>

        <input type="range"
               className="anthem-volume"
               aria-label={t("anthem.volume")}
               min={0} max={1} step={0.05}
               value={settings.enabled && settings.anthem.on ? settings.anthem.volume : 0}
               disabled={!recorded}
               onChange={(event) => setVolume(Number(event.target.value))}/>

        <div ref={progress} className="anthem-progress" aria-hidden="true"/>
    </section>
}

function subtitle(anthem: Anthem, playing: boolean, country: string): string {
    if (anthem.title === undefined) return t("anthem.noRecording")
    if (playing && !anthem.unlocked) return t("anthem.clickToStart")
    return t("anthem.byNavyBand", {country})
}
