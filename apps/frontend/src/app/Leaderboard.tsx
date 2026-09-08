import "./Leaderboard.css"
import {useId} from "react";
import {Country} from "../domain/countries.ts";
import {LeaderboardEntry} from "../domain/leaderboard.ts";
import {truncate} from "./truncate.ts";

type LeaderboardProps = {
    tilesCount: number,
    data: LeaderboardEntry[],
    /** The country the player holds, marked in the table so they can find themselves. */
    highlight?: Country,
}

/** Names wider than this overflow the panel on a phone. */
const NAME_MAX_LENGTH = 18

/**
 * The table, and nothing else: collapsing is the card's job now, not this
 * component's. It used to own a "Hide" button that sat above an unlabelled
 * table, so the only thing on screen naming what would be hidden was the button
 * that hid it.
 */
export default function Leaderboard(props: LeaderboardProps) {
    const titleId = useId()

    return <section className="leaderboard" aria-labelledby={titleId}>
        <h2 className="menu-section-title" id={titleId}>Leaderboard</h2>

        <div className="leaderboard-table-container">
            <table className="leaderboard-table">
                <thead>
                <tr>
                    <th className="leaderboard-table-head leaderboard-table-rank" scope="col">#</th>
                    <th className="leaderboard-table-head" scope="col">Country</th>
                    <th className="leaderboard-table-head leaderboard-table-number" scope="col">Tiles</th>
                    <th className="leaderboard-table-head leaderboard-table-number" scope="col">Share</th>
                </tr>
                </thead>

                <tbody>
                {props.data.map((entry, index) => {
                    const isPlayer = entry.country.code === props.highlight?.code
                    return <tr key={entry.country.code}
                               className={isPlayer ? "leaderboard-entry leaderboard-entry-player" : "leaderboard-entry"}
                               aria-current={isPlayer ? "true" : undefined}>
                        <td className="leaderboard-entry-index">{index + 1}</td>
                        <td>{truncate(entry.country.name, NAME_MAX_LENGTH)}</td>
                        <td className="leaderboard-table-number leaderboard-table-tiles">{entry.tiles}</td>
                        <td className="leaderboard-table-number">
                            {(entry.tiles / props.tilesCount * 100).toFixed(2)}
                        </td>
                    </tr>
                })}
                </tbody>
            </table>
        </div>
    </section>
}
