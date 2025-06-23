export function other01({ SomeMap }: { SomeMap: {items: string[]} }) {
    return (
        <div class="foo">
            <h1>Hi!</h1>
            <p>Here is a list of {SomeMap.items.length} items:</p>
            <ul>
                {SomeMap.items.map((item) => (
                    <li>{item}</li>
                ))}
            </ul>
        </div>
    );
}