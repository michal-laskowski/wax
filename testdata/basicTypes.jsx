export function basicTypes() {
    return (
        <>
            
            <in-inner>
                {/* boolean, null, undefined are ignored */}
                <sample-01>{false}x{true}</sample-01>
                <sample-02>{null}</sample-02>
                <sample-03>{undefined}</sample-03>
                <sample-04>{0}</sample-04>
                <sample-05>{432}</sample-05>
                <sample-06>{NaN}</sample-06>
                <sample-07>{true}</sample-07>
                <sample-08>{Infinity}</sample-08>
                <sample-09>{-Infinity}</sample-09>
                {/* for array values will be processed */}
                <sample-10>{[1, false, NaN, "some_string"]}</sample-10>
                <sample-11>{123456789123456789n}</sample-11>
                <sample-12>{-1}x{0}x{1}</sample-12>
                <sample-13>{-1} {0} {1}</sample-13>
                {/* date will be rendered in ISO format */}
                <sample-14>{new Date("1914-12-20T08:00+0000")}</sample-14>
            </in-inner>
            <in-attr>
                <sample-01 v={false} v2={true}/>
                <sample-02 v={null}/>
                <sample-03 v={undefined}/>
                <sample-04 v={0}/>
                <sample-05 v={432}/>
                <sample-06 v={NaN}/>
                <sample-07 v={true}/>
                <sample-08 v={Infinity}/>
                <sample-09 v={-Infinity}/>
                <sample-10 v={[1, false, NaN, "some_string"]}/>
                <sample-11 v={123456789123456789n}/>
                <sample-14 v={new Date("1914-12-20T08:00+0000")}/>
            </in-attr>
        </>
    );
}