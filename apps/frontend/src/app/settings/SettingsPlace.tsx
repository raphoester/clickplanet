import {DisplaySettings} from "../../domain/displaySettings.ts";
import Switch from "../components/Switch.tsx";
import SoundSettingsPanel, {SoundSettingsPanelProps} from "../sound/SoundSettingsPanel.tsx";
import "./SettingsPlace.css"

export type SettingsPlaceProps = {
    display?: {settings: DisplaySettings, onChange: (settings: DisplaySettings) => void},
    sound?: SoundSettingsPanelProps,
}

export default function SettingsPlace({display, sound}: SettingsPlaceProps) {
    return <div className="settings">
        {display && <ul className="switch-list panel-box">
            <li>
                <Switch label="Big country flags"
                        checked={display.settings.mapView === "flags"}
                        onChange={(on) => display.onChange({...display.settings, mapView: on ? "flags" : "tiles"})}/>
            </li>
            <li>
                <Switch label="HD graphics"
                        checked={display.settings.rendering === "sharp"}
                        onChange={(on) => display.onChange({...display.settings, rendering: on ? "sharp" : "plain"})}/>
            </li>
        </ul>}
        {sound && <SoundSettingsPanel {...sound}/>}
    </div>
}
