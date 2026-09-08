import {useEffect, useRef, useState} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry} from "../domain/leaderboard.ts";
import Leaderboard from "./Leaderboard.tsx";
import About from "./About.tsx";
import CountryPicker from "./CountryPicker.tsx";
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
 * The sticky card, which drills down rather than growing. A panel takes the
 * place of both the leaderboard and the buttons that opened it, and its own
 * close button walks back up; stacking them instead ran the card down the whole
 * page and left five buttons piled at the bottom.
 *
 * That is also why the open panel lives here rather than in each button: there
 * is one content slot, so at most one panel can be in it.
 */
export default function Menu(props: MenuProps) {
    const [openPanel, setOpenPanel] = useState<PanelId | null>(null)
    const close = () => setOpenPanel(null)

    /**
     * Drilling in unmounts the button that was clicked, so there is no element
     * left to hand focus back to when the panel closes. Remember which button
     * it was and focus it once the actions are on screen again, otherwise a
     * keyboard user is returned to the top of the document.
     */
    const cameFrom = useRef<PanelId | null>(null)
    const countryButton = useRef<HTMLButtonElement>(null)
    const aboutButton = useRef<HTMLButtonElement>(null)

    const open = (panel: PanelId) => {
        cameFrom.current = panel
        setOpenPanel(panel)
    }

    useEffect(() => {
        if (openPanel !== null || cameFrom.current === null) return
        const returningTo = cameFrom.current
        cameFrom.current = null
        const button = returningTo === "country" ? countryButton : aboutButton
        button.current?.focus()
    }, [openPanel])

    return <div className="menu">
        <div className="menu-header">
            <img alt="ClickPlanet logo"
                 src="/static/logo.svg"
                 width="56px"
                 height="56px"/>
            <h1>ClickPlanet</h1>
        </div>

        {openPanel === null && <>
            <Leaderboard data={props.leaderboard} tilesCount={props.tilesCount}/>

            <div className="menu-actions">
                <button ref={countryButton} className="button" onClick={() => open("country")}>
                    {props.country.name}
                </button>
                <button ref={aboutButton} className="button" onClick={() => open("about")}>
                    About
                </button>
                <DiscordButton/>
            </div>
        </>}

        {openPanel === "country" && <MenuPanel title="Country" onClose={close}>
            <CountryPicker country={props.country} setCountry={props.setCountry}/>
        </MenuPanel>}

        {openPanel === "about" && <MenuPanel title="About" onClose={close} closeButtonText="Back">
            <About/>
        </MenuPanel>}
    </div>
}
