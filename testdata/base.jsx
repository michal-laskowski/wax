export function base() {
    return <>
        <a href="test">a link</a>
        <ul>
            {[1, 2].map((li) => (
                <li>{li}</li>
            ))}
        </ul>
        <button onclick="doSomething"></button>
        <div class="class-a"></div>
        
        <script
            src="jquery.js"
            integrity="sha256-123="
            crossorigin="anonymous"
        ></script>

        <svg width="100" height="100" viewBox="0 0 100 100">
            <circle
                cx="50"
                cy="50"
                r="40"
                stroke="green"
                stroke-width="4"
                fill="yellow"
            />
        </svg>
    </>
}
