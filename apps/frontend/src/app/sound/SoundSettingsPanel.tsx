import {SoundSettings, SWITCHES, SwitchName} from "../../domain/soundSettings.ts";
import Switch from "../components/Switch.tsx";
import {useSoundSettings} from "./soundSettingsStore.ts";
import {t} from "../../i18n/page.ts";
import {Key} from "../../i18n/translator.ts";
import "./SoundSettingsPanel.css"

const LABELS: Record<SwitchName, Key> = {
    click: "sound.tileClick",
    refused: "sound.refusedClick",
    bonusSpawn: "sound.bonusAppears",
    bonusCaught: "sound.bonusCaught",
    bomb: "sound.bomb",
    fortify: "sound.fortified",
    chat: "sound.chatMessage",
    quiz: "sound.quiz",
    title: "sound.titleUnlocked",
}

export type SoundSettingsPanelProps = {
    preview: (name: SwitchName) => void
}

export default function SoundSettingsPanel({preview}: SoundSettingsPanelProps) {
    const {settings, setSettings} = useSoundSettings()
    const setEnabled = (enabled: boolean) => {
        setSettings({...settings, enabled})
        if (enabled) preview("click")
    }

    const setSound = (name: SwitchName, on: boolean) => {
        setSettings({...settings, sounds: {...settings.sounds, [name]: on}})
        if (on) preview(name)
    }

    const setAnthem = (anthem: SoundSettings["anthem"]) => setSettings({...settings, anthem})

    const listClass = settings.enabled ? "switch-list panel-box" : "switch-list panel-box sound-settings-list--off"

    return <div className="sound-settings">
        <Switch label={t("sound.sound")} checked={settings.enabled} onChange={setEnabled} main/>

        <div className={listClass}>
            <Switch label={t("sound.anthem")}
                    checked={settings.anthem.on}
                    disabled={!settings.enabled}
                    onChange={(on) => setAnthem({...settings.anthem, on})}/>
            <label className="sound-volume">
                <span>{t("sound.anthemVolume")}</span>
                <input type="range"
                       className="sound-volume-input"
                       min={0} max={1} step={0.05}
                       value={settings.anthem.volume}
                       disabled={!settings.enabled || !settings.anthem.on}
                       onChange={(event) => setAnthem({...settings.anthem, volume: Number(event.target.value)})}/>
            </label>
        </div>

        <ul className={listClass}>
            {SWITCHES.map((name) => <li key={name}>
                <Switch label={t(LABELS[name])}
                        checked={settings.sounds[name]}
                        disabled={!settings.enabled}
                        onChange={(on) => setSound(name, on)}/>
            </li>)}
        </ul>
    </div>
}
