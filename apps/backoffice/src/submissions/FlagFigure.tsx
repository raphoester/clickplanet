import {useState} from "react"
import {Picture} from "../gen/grpc/flags/v1/flags_pb.ts"
import {TILE_SIZES} from "./submissions.ts"

type Props = {
    picture: Picture | undefined
    // What the picture is of, for a screen reader.
    name: string
}

// A flag's picture as the server drew it, at its own size up to the page's width, then as the tiles show it.
export function FlagFigure({picture, name}: Props) {
    const [broken, setBroken] = useState(false)

    return (
        <figure className="submission-figure">
            {picture === undefined && <p className="muted">The picture was deleted.</p>}
            {picture !== undefined && broken && <p className="action-refused">The picture did not load: its link may have run out. Reload the list.</p>}
            {picture !== undefined && !broken && (
                <>
                    <img
                        className="submission-picture"
                        src={picture.url}
                        width={picture.width || undefined}
                        height={picture.height || undefined}
                        alt={`The flag of ${name}`}
                        onError={() => setBroken(true)}
                    />
                    <div className="on-tiles">
                        {TILE_SIZES.map(size => (
                            <img
                                key={size}
                                className="on-tile"
                                src={picture.url}
                                width={size}
                                height={size}
                                alt={`The flag of ${name} on a tile ${size} px wide`}
                            />
                        ))}
                        <span className="hint">{picture.width} × {picture.height}</span>
                    </div>
                </>
            )}
        </figure>
    )
}
