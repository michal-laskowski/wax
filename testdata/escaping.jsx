
export function escaping(){
    return (<>
        <in-inner>
            <sample-01>{'"<>&"\''}</sample-01>
            <sample-02>{"'<>&\"'"}</sample-02>
            <sample-03><script>alert('sample-03')</script></sample-03>
            <sample-04>{"<script>alert('in-inner-sample-04')</script>"}</sample-04>
            <sample-05>{`<script>alert("in-inner-sample-05")</script>`}</sample-05>
            <sample-06 title={"'\"<>&"} style={{ textAlign: "'\"<>&" }}>{"'\"<>&"}</sample-06>
            <sample-07>{"<script type='' src=\"\"></script>"}</sample-07>
        </in-inner>
        <in-attr>
            <sample-01 v={'"<>&"\''} />
            <sample-02 v={"'<>&\"'"} />
            <sample-03 xstyle={"\"&<>'"} />
            <sample-04 style={{ backgroundColor: "\"&<>'" }} />
            <sample-05 class={"\"&<>'"} />
            <sample-06 class={'test:1" xss="false'} />
            <sample-07 onclick={`<script>alert("xdd")</script>`}>in-attr-sample-07</sample-07>
            <sample-08 v="'bar'" />
            <sample-09 v='"bar\"' />
            <sample-10 v="bar\`" />
            <sample-11 v={"'bar'"} />
            <sample-12 v={'"bar\`\"\''} />
            <sample-13 v={"bar\`\"\'"} />
        </in-attr>
    </>)
}