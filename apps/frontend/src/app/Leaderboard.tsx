import "./Leaderboard.css"
import {LeaderboardEntry} from "../domain/leaderboard.ts";
import {useState} from "react";
import {truncate} from "./truncate.ts";


type LeaderboardProps = {
    tilesCount: number,
    data: LeaderboardEntry[]
}

/** Names wider than this overflow the panel on a phone. */
const NAME_MAX_LENGTH = 18

export default function Leaderboard(props: LeaderboardProps) {
    const [isOpen, setIsOpen] = useState(true)
    const toggleLeaderboard = () => {
        setIsOpen(!isOpen)
    }

    return <div className="leaderboard">
        <div className="leaderboard-header">
            <img alt="ClickPlanet logo"
                 src="/static/logo.svg"
                 width="56px"
                 height="56px"/>
            <h1>ClickPlanet</h1>
        </div>
        <div className="leaderboard-expand">
            <button className="button button-leaderboard"
                    onClick={toggleLeaderboard}>{isOpen ? "Hide" : "Leaderboard"}</button>
        </div>
        {isOpen &&
            <div className="leaderboard-table-container">
                <table className="leaderboard-table">
                    <thead>
                    <tr>
                        <th></th>
                        <th colSpan={3}>🌍</th>
                        <th className="leaderboard-table-number leaderboard-table-tiles">⚪️</th>
                        <th className="leaderboard-table-number">%</th>
                    </tr>
                    </thead>

                    <tbody>
                    {props.data.map((entry, index) => {
                        return <tr key={index} className="leaderboard-entry">
                            <td className="leaderboard-entry-index">{index + 1}.</td>
                            <td colSpan={3}>{truncate(entry.country.name, NAME_MAX_LENGTH)}</td>
                            <td className="leaderboard-table-number leaderboard-table-tiles">{entry.tiles}</td>
                            <td className="leaderboard-table-number">
                                {(entry.tiles / props.tilesCount * 100).toFixed(2)}
                            </td>
                        </tr>
                    })}
                    </tbody>
                </table>
            </div>
        }
    </div>
}