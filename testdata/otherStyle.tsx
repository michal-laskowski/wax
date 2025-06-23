export function otherStyle() {
    const style: CSSStyleDeclaration = {
        color: "red",
        //@ts-expect-error
        not: "defined"
    }

    return (<>
        <sample-01 style={ { color: "red", border: "none" } } ></sample-01>
        <sample-02 style={ { backgroundColor: "red" } } ></sample-02>
        <sample-03 style={ { "background-color": "green" } }></sample-03>
        <sample-04 style="background-color: blue;" ></sample-04>
        <sample-05 style={ { "--foo": "1", "--foo-bar": "2" } } ></sample-05>
        <sample-06 style={ { 'background-image': 'url("example.png")' } } ></sample-06>
        <sample-07 style={ { "background-image": "url(\"example.png\")" } } ></sample-07>
        <sample-08 style={ { "background-image": "url('example.png')" } } ></sample-08>
        <sample-09 style={ { "background-image": "url('example.png')" } } ></sample-09>
        <sample-10 style={ { "content": "'\"foo bar'" } }></sample-10>
        <sample-11 style={ style } ></sample-11>
        <sample-12 style={ { backgroundColor: undefined } }>{/*not a string, will skip property*/}</sample-12>
        <sample-13 style={ { backgroundColor: null } }>{/*not a string, will skip property*/}</sample-13>
        <sample-14 style={ { backgroundColor: false } }>{/*not a string, will skip property*/}</sample-14>
        <sample-15 style={ { backgroundColor: true } }>{/*not a string, will skip property*/}</sample-15>
        <sample-16 style={ { width: 100 } }>{/*not a string, will skip property*/}</sample-16>
        <sample-17 style={ { color: "" } }>{/*empty string, will skip property*/}</sample-17>
        <sample-18 style={ {} }>{/*on empty style, attribute will be skipped*/}</sample-18>
    </>)
}