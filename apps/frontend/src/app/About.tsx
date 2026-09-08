import "./About.css"

/** The contents of the About dialog; the chrome, and the coffee, are Modal's. */
export default function About() {
    return <>
        <h3>The ultimate world war</h3>
        <h4>It’s like Pixel Wars but way more epic</h4>
        <p>ClickPlanet is a virtual battleground where you conquer territories for a country, click after
            click. Out-click rival nations, and dominate the map. Every territory is yours, until someone
            takes it back!</p>
        <div className="about-author">
            <p className="menu-label">Created by</p>
            <div className="center-align">
                <img alt="Raphaël Oester"
                     src="/static/raphael.jpeg"
                     className="about-photo"/>
            </div>
            <h3>Raphaël Oester</h3>
            <p className="center-align">Freelance developer, open to new opportunities</p>
            <div className="about-social">
                <a target="_blank" rel="noopener noreferrer" href="https://www.linkedin.com/in/raphael-oester/"><b>in</b></a>
                <a target="_blank" rel="noopener noreferrer" href="https://x.com/raphael_oester"><b>X</b></a>
                <a target="_blank" rel="noopener noreferrer" href="https://github.com/raphoester"><b>GitHub</b></a>
            </div>
        </div>
    </>
}
