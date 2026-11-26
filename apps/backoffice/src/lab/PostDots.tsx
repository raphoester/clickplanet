import {CSSProperties} from "react"
import {Dot} from "./chart.ts"

export function PostDots({dots, shown}: {dots: Dot[], shown: string | undefined}) {
    return (
        <>
            {dots.map(dot => (
                <circle key={dot.key} className={dot.young ? "lab-dot-young" : "lab-dot"}
                        style={{"--arm": dot.color} as CSSProperties} cx={dot.x} cy={dot.y} r={4}/>
            ))}
            {dots.filter(dot => dot.post.postId === shown).map(dot => (
                <circle key={`shown-${dot.key}`} className="lab-dot-shown" cx={dot.x} cy={dot.y} r={7}/>
            ))}
        </>
    )
}
