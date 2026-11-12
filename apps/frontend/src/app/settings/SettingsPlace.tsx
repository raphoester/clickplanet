import {useState} from "react";
import {MapView} from "../../domain/displaySettings.ts";
import {FeedbackBackend, PageContext} from "../../backends/support.ts";
import About from "../About.tsx";
import {DiscordOffer} from "../alerts/useDiscordAlerts.ts";
import BuyMeACoffee from "../components/BuyMeACoffee.tsx";
import Choices from "../components/Choices.tsx";
import {InfoIcon, MailIcon} from "../components/icons.tsx";
import Modal from "../components/Modal.tsx";
import Switch from "../components/Switch.tsx";
import SoundSettingsPanel, {SoundSettingsPanelProps} from "../sound/SoundSettingsPanel.tsx";
import FeedbackModal from "../support/FeedbackModal.tsx";
import {pageContextOf} from "../support/pageContext.ts";
import {NO_GRAPHICS} from "../viewer/graphicsReport.ts";
import {ON_CRAZYGAMES} from "../../platform.ts";
import {useDisplaySettings} from "./displaySettingsStore.ts";
import LanguagePicker from "./LanguagePicker.tsx";
import {t} from "../../i18n/page.ts";
import {Key} from "../../i18n/translator.ts";
import "./SettingsPlace.css"

export type SettingsPlaceProps = {
    sound?: SoundSettingsPanelProps,
    support: FeedbackBackend,
    pageContext?: () => PageContext,
    discord?: DiscordOffer,
}

const pageAlone = () => pageContextOf(window, NO_GRAPHICS, false)

const MAP_VIEWS: {value: MapView, label: Key}[] = [
    {value: "big", label: "settings.mapViewBig"},
    {value: "split", label: "settings.mapViewSplit"},
    {value: "tiles", label: "settings.mapViewOff"},
]

export default function SettingsPlace({sound, support, pageContext = pageAlone, discord}: SettingsPlaceProps) {
    const [open, setOpen] = useState<"feedback" | "about">()
    const close = () => setOpen(undefined)
    const display = useDisplaySettings()

    return <div className="settings">
        <ul className="switch-list panel-box">
            <li className="settings-choices">
                <LanguagePicker/>
            </li>
            <li className="settings-choices">
                <Choices label={t("settings.countryFlags")}
                         options={MAP_VIEWS.map(({value, label}) => ({value, label: t(label)}))}
                         picked={display.mapView}
                         onPick={display.setMapView}/>
            </li>
            <li>
                <Switch label={t("settings.hdGraphics")}
                        checked={display.rendering === "sharp"}
                        onChange={(on) => display.setRendering(on ? "sharp" : "soft")}/>
            </li>
        </ul>
        {sound && <SoundSettingsPanel {...sound}/>}

        <div className="menu-tiles settings-foot">
            <button type="button"
                    className="panel-box menu-tile"
                    onClick={() => setOpen("feedback")}>
                <MailIcon size={22}/>
                <span>{t("settings.feedback")}</span>
            </button>
            <button type="button"
                    className="panel-box menu-tile"
                    onClick={() => setOpen("about")}>
                <InfoIcon size={22}/>
                <span>{t("settings.about")}</span>
            </button>
        </div>

        {open === "about" && <Modal title={t("settings.aboutTitle")}
                                    footer={ON_CRAZYGAMES ? undefined : <BuyMeACoffee/>}
                                    onClose={close}>
            <About discord={discord}/>
        </Modal>}

        {open === "feedback" && <FeedbackModal support={support}
                                               pageContext={pageContext}
                                               onClose={close}/>}
    </div>
}
