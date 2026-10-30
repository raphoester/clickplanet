import {useState} from "react";
import {DisplaySettings, MapView} from "../../domain/displaySettings.ts";
import {FeedbackBackend, PageContext} from "../../backends/support.ts";
import About from "../About.tsx";
import BuyMeACoffee from "../components/BuyMeACoffee.tsx";
import Choices, {Choice} from "../components/Choices.tsx";
import {InfoIcon, MailIcon} from "../components/icons.tsx";
import Modal from "../components/Modal.tsx";
import Switch from "../components/Switch.tsx";
import SoundSettingsPanel, {SoundSettingsPanelProps} from "../sound/SoundSettingsPanel.tsx";
import FeedbackModal from "../support/FeedbackModal.tsx";
import {pageContextOf} from "../support/pageContext.ts";
import {NO_GRAPHICS} from "../viewer/graphicsReport.ts";
import {ON_CRAZYGAMES} from "../../platform.ts";
import "./SettingsPlace.css"

export type SettingsPlaceProps = {
    display?: {settings: DisplaySettings, onChange: (settings: DisplaySettings) => void},
    sound?: SoundSettingsPanelProps,
    support?: FeedbackBackend,
    pageContext?: () => PageContext,
}

const pageAlone = () => pageContextOf(window, NO_GRAPHICS, false)

const MAP_VIEWS: Choice<MapView>[] = [
    {value: "big", label: "Big"},
    {value: "split", label: "Split"},
    {value: "tiles", label: "Off"},
]

export default function SettingsPlace({display, sound, support, pageContext = pageAlone}: SettingsPlaceProps) {
    const [open, setOpen] = useState<"feedback" | "about">()
    const close = () => setOpen(undefined)

    return <div className="settings">
        {display && <ul className="switch-list panel-box">
            <li className="settings-choices">
                <Choices label="Country flags"
                         options={MAP_VIEWS}
                         picked={display.settings.mapView}
                         onPick={(mapView) => display.onChange({...display.settings, mapView})}/>
            </li>
            <li>
                <Switch label="HD graphics"
                        checked={display.settings.rendering === "sharp"}
                        onChange={(on) => display.onChange({...display.settings, rendering: on ? "sharp" : "plain"})}/>
            </li>
        </ul>}
        {sound && <SoundSettingsPanel {...sound}/>}

        <div className="menu-tiles settings-foot">
            {support && <button type="button"
                                className="panel-box menu-tile"
                                onClick={() => setOpen("feedback")}>
                <MailIcon size={22}/>
                <span>Feedback</span>
            </button>}
            <button type="button"
                    className="panel-box menu-tile"
                    onClick={() => setOpen("about")}>
                <InfoIcon size={22}/>
                <span>About</span>
            </button>
        </div>

        {open === "about" && <Modal title="About ClickPlanet"
                                    footer={ON_CRAZYGAMES ? undefined : <BuyMeACoffee/>}
                                    onClose={close}>
            <About/>
        </Modal>}

        {open === "feedback" && support && <FeedbackModal support={support}
                                                          pageContext={pageContext}
                                                          onClose={close}/>}
    </div>
}
