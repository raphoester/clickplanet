import {SoundSettings, SWITCHES, SwitchName} from "../../domain/soundSettings.ts";
import Switch from "../components/Switch.tsx";
import "./SoundSettingsPanel.css"

const LABELS: Record<SwitchName, string> = {
    click: "Tile click",
    refused: "Refused click",
    bonusSpawn: "Bonus box appears",
    bonusCaught: "Bonus box caught",
    bomb: "Bomb explosion",
    chat: "Chat message",
    quiz: "Quiz",
    title: "Title unlocked",
}

export type SoundSettingsPanelProps = {
    settings: SoundSettings
    onChange: (settings: SoundSettings) => void
    preview: (name: SwitchName) => void
}

export default function SoundSettingsPanel({settings, onChange, preview}: SoundSettingsPanelProps) {
    const setEnabled = (enabled: boolean) => {
        onChange({...settings, enabled})
        if (enabled) preview("click")
    }

    const setSound = (name: SwitchName, on: boolean) => {
        onChange({...settings, sounds: {...settings.sounds, [name]: on}})
        if (on) preview(name)
    }

    const setAnthem = (anthem: SoundSettings["anthem"]) => onChange({...settings, anthem})

    const listClass = settings.enabled ? "switch-list panel-box" : "switch-list panel-box sound-settings-list--off"

    return <div className="sound-settings">
        <Switch label="Sound" checked={settings.enabled} onChange={setEnabled} main/>

        <div className={listClass}>
            <Switch label="Leader's national anthem"
                    checked={settings.anthem.on}
                    disabled={!settings.enabled}
                    onChange={(on) => setAnthem({...settings.anthem, on})}/>
            <label className="sound-volume">
                <span>Anthem volume</span>
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
                <Switch label={LABELS[name]}
                        checked={settings.sounds[name]}
                        disabled={!settings.enabled}
                        onChange={(on) => setSound(name, on)}/>
            </li>)}
        </ul>
    </div>
}
