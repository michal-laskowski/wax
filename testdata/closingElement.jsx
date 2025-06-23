export function closingElement() {
    return (
        <>
            <area />
            <area></area>
            <base />
            <br />
            <col />
            <command />
            <embed />
            <hr />
            <hr></hr>
            <img />
            <input />
            <keygen />
            <link />
            <meta />
            <param />
            <source />
            <track />
            <wbr />
            {/* Not void elements */}
            <div />
            <span />
            <p />
            <custom-element></custom-element>
            <custom-element />
        </>
    );
}