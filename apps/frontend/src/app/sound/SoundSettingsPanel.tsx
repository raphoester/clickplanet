import {useId} from "react";
import {SoundSettings, SWITCHES, SwitchName} from "../../domain/soundSettings.ts";
import "./SoundSettingsPanel.css"

const LABELS: Record<SwitchName, string> = {
    click: "Tile click",
    refused: "Refused click",
    bonusSpawn: "Bonus box appears",
    bonusCaught: "Bonus box caught",
    bomb: "Bomb explosion",
    chat: "Chat message",
}

export type SoundSettingsPanelProps = {
    settings: SoundSettings
    onChange: (settings: SoundSettings) => void
    /** Plays a sound regardless of the settings, so a switch turned on is heard. */
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

    return <div className="sound-settings">
        <Switch label="Sound" checked={settings.enabled} onChange={setEnabled} main/>

        <div className={settings.enabled ? "sound-settings-list" : "sound-settings-list sound-settings-list--off"}>
            <Switch label="Leader's national anthem"
                    checked={settings.anthem.on}
                    disabled={!settings.enabled}
                    onChange={(on) => setAnthem({...settings.anthem, on})}/>
            <label className="sound-volume">
                <span className="sound-switch-label">Anthem volume</span>
                <input type="range"
                       className="sound-volume-input"
                       min={0} max={1} step={0.05}
                       value={settings.anthem.volume}
                       disabled={!settings.enabled || !settings.anthem.on}
                       onChange={(event) => setAnthem({...settings.anthem, volume: Number(event.target.value)})}/>
            </label>
        </div>

        <ul className={settings.enabled ? "sound-settings-list" : "sound-settings-list sound-settings-list--off"}>
            {SWITCHES.map((name) => <li key={name}>
                <Switch label={LABELS[name]}
                        checked={settings.sounds[name]}
                        disabled={!settings.enabled}
                        onChange={(on) => setSound(name, on)}/>
            </li>)}
        </ul>
    </div>
}

type SwitchProps = {
    label: string
    checked: boolean
    disabled?: boolean
    main?: boolean
    onChange: (checked: boolean) => void
}

function Switch({label, checked, disabled, main, onChange}: SwitchProps) {
    const id = useId()
    return <label className={main ? "sound-switch sound-switch--main" : "sound-switch"} htmlFor={id}>
        <span className="sound-switch-label">{label}</span>
        <input id={id}
               type="checkbox"
               role="switch"
               className="sound-switch-input"
               checked={checked}
               disabled={disabled}
               onChange={(event) => onChange(event.target.checked)}/>
    </label>
}
