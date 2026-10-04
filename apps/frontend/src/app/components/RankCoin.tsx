export default function RankCoin({rank, you}: {rank: number, you: boolean}) {
    if (you) return <span className="coin coin-you">{rank}</span>
    if (rank <= 3) return <span className={`coin coin-${rank}`}>{rank}</span>
    return <span className="leaderboard-rank">{rank}</span>
}
