export function basicTypes_from_model(p) {
    return (
        <>
            <sample-01>{p.SomeStringArr}</sample-01>
            <sample-02>{p.SomeStringPtrArr}</sample-02>
            <sample-03>{p.PtrArr}</sample-03>
            <sample-04>{p.PtrArrPtr}</sample-04>
            <sample-05>{p.AliasToString}</sample-05>
            <sample-05a>
                {typeof p.AliasToString} - type alias will be object in goja
            </sample-05a>
            <sample-05b>
                {typeof p.AliasToString.SomeAliasTrueMethod}
            </sample-05b>
            <sample-05c>
                {typeof p.AliasToString.SomeAliasFalseMethod}
            </sample-05c>
            <sample-06>{p.Ballance}</sample-06>
            <sample-07>{p.Deposit}</sample-07>
            <sample-08>{p.OtherDummySimple}</sample-08>
            <sample-09>{p.OtherDummyMaps}</sample-09>
            <sample-10>{p.OtherDummySimpleGeneric}</sample-10>
            <sample-11>{p.OtherDummyBasicTypes}</sample-11>
            <sample-12>{p.DummySimple}</sample-12>
            <sample-13>{p.GoPublicThatDoesNotExist}</sample-13>
            <sample-13>{p.goPrivateField}</sample-13>
        </>
    );
}
