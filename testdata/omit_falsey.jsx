export function omit_falsey() {
    return (<>
        <in-inner>
            <sample-01>{null}</sample-01>
            <sample-02>{undefined}</sample-02>
            <sample-03>{false}</sample-03>
            <sample-04>{0}</sample-04>
            <sample-99>{null}|{undefined}|{false}|{0}</sample-99>
        </in-inner>
        <in-attr>
            <sample-01 v={null} />
            <sample-02 v={undefined}  />
            <sample-03 v={false}  />
            <sample-04 v={0} />
            <sample-99 v1={null} v2={undefined} v3={false} v4={0} />
        </in-attr>
    </>)
}