export function attribute_boolean_enumerable() {
    return (<>
        <attr-known>
            <sample-01 draggable={true} />
            <sample-02 draggable={false} />
            <sample-03 spellcheck={true} />
            <sample-04 spellcheck={false} />
            <sample-05 draggable={"true"} />
            <sample-06 draggable={"false"} />
            <sample-07 spellcheck={"true"} />
            <sample-08 spellcheck={"false"} />
        </attr-known>
        <attr-aria>
            <sample-01 aria-checked={true} />
            <sample-02 aria-checked={false} />
            <sample-03 aria-selected={true} />
            <sample-04 aria-selected={false} />
        </attr-aria>
    </>)
}