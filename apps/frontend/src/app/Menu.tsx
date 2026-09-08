import {useState} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry} from "../domain/leaderboard.ts";
import Leaderboard from "./Leaderboard.tsx";
import About from "./About.tsx";
import CountryPicker from "./CountryPicker.tsx";
import MenuButton from "./components/MenuButton.tsx";
import MenuPanel from "./components/MenuPanel.tsx";
import DiscordButton from "./components/DiscordButton.tsx";
import "./Menu.css"

type PanelId = "country" | "about"

export type MenuProps = {
    country: Country,
    setCountry: (country: Country) => void,
    leaderboard: LeaderboardEntry[],
    tilesCount: number,
}

/**
 * The sticky card. Which panel is open lives here rather than in each button,
 * because the panels expand inside the card: two open at once would push it
 * past the viewport, so opening one closes the other.
 */
export default function Menu(props: MenuProps) {
    const [openPanel, setOpenPanel] = useState<PanelId | null>(null)
    const toggle = (panel: PanelId) => setOpenPanel((open) => open === panel ? null : panel)
    const close = () => setOpenPanel(null)

    return <div className="menu">
        <Leaderboard data={props.leaderboard} tilesCount={props.tilesCount}/>

        <div className="menu-actions">
            <MenuButton
                text={props.country.name}
                expanded={openPanel === "country"}
                onClick={() => toggle("country")}
            />
            <MenuButton
                text="About"
                expanded={openPanel === "about"}
                onClick={() => toggle("about")}
            />
            <DiscordButton/>
        </div>

        {openPanel === "country" && <MenuPanel title="Country" onClose={close}>
            <CountryPicker country={props.country} setCountry={props.setCountry}/>
        </MenuPanel>}

        {openPanel === "about" && <MenuPanel title="About" onClose={close} closeButtonText="Back">
            <About/>
        </MenuPanel>}
    </div>
}
