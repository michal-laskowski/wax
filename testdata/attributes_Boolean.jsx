export function attributes_Boolean() {
    return (
        <>
            <input checked={true}></input>
            <input checked={false}></input>
            <input disabled={true}></input>
            <input disabled={false}></input>
            <p hidden translate></p>
            <p hidden={false} translate={false}></p>
            <form novalidate></form>
        </>
    );
}
