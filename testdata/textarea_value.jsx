export function textarea_value() {
    const value = "<div>test area expected value</div>";
    return (
        <>
            <textarea value="abc" />
            <textarea value={`a&b"c`} />
            <textarea value="true" />
            <textarea value={value} />
            <textarea value={true} />
            <textarea value="" />
            <textarea value={""} />
            <textarea value={"    "} />
        </>
    );
}