import {useState} from "react";
import {DisplaySettings} from "../../domain/displaySettings.ts";
import {FeedbackBackend, PageContext} from "../../backends/support.ts";
import About from "../About.tsx";
import BuyMeACoffee from "../components/BuyMeACoffee.tsx";
import {InfoIcon, MailIcon, ReportIcon} from "../components/icons.tsx";
import Modal from "../components/Modal.tsx";
import Switch from "../components/Switch.tsx";
import SoundSettingsPanel, {SoundSettingsPanelProps} from "../sound/SoundSettingsPanel.tsx";
import FeedbackModal from "../support/FeedbackModal.tsx";
import {pageContextOf} from "../support/pageContext.ts";
import {NO_GRAPHICS} from "../viewer/graphicsReport.ts";
import "./SettingsPlace.css"

export type SettingsPlaceProps = {
    display?: {settings: DisplaySettings, onChange: (settings: DisplaySettings) => void},
    sound?: SoundSettingsPanelProps,
    support?: FeedbackBackend,
    pageContext?: () => PageContext,
    // Closes what is open and lets the player tap the drawing on the globe.
    onReportPlace?: () => void,
}

const pageAlone = () => pageContextOf(window, NO_GRAPHICS, false)

export default function SettingsPlace({display, sound, support, pageContext = pageAlone, onReportPlace}: SettingsPlaceProps) {
    const [open, setOpen] = useState<"feedback" | "about">()
    const close = () => setOpen(undefined)

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

        <div className="menu-tiles settings-foot">
            {support && <button type="button"
                                className="panel-box menu-tile"
                                onClick={() => setOpen("feedback")}>
                <MailIcon size={22}/>
                <span>Feedback</span>
            </button>}
            {onReportPlace && <button type="button"
                                      className="panel-box menu-tile"
                                      onClick={onReportPlace}>
                <ReportIcon size={22}/>
                <span>Report a drawing</span>
            </button>}
            <button type="button"
                    className="panel-box menu-tile"
                    onClick={() => setOpen("about")}>
                <InfoIcon size={22}/>
                <span>About</span>
            </button>
        </div>

        {open === "about" && <Modal title="About ClickPlanet"
                                    footer={<BuyMeACoffee/>}
                                    onClose={close}>
            <About/>
        </Modal>}

        {open === "feedback" && support && <FeedbackModal support={support}
                                                          pageContext={pageContext}
                                                          onClose={close}/>}
    </div>
}
