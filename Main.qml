import QtQuick 2.0

Loader {
    anchors.fill: parent

    function random(arr) {
        return arr[Math.floor(Math.random() * arr.length)]
    }

    // 🎨 Core pixel set (main vibe)
    function pixelCore() {
        return random([
            "themes/pixel-coffee/Main.qml",
            "themes/pixel-dusk-city/Main.qml",
            "themes/pixel-emerald/Main.qml",
            "themes/pixel-night-city/Main.qml",
            "themes/pixel-rainyroom/Main.qml",
            "themes/pixel-skyscrapers/Main.qml"
        ])
    }

    // 🌙 Alternate pixel set (rare flavor)
    function pixelAlt() {
        return random([
            "themes/pixel-hollowknight/Main.qml",
            "themes/pixel-munchlax/Main.qml"
        ])
    }

    // 🧠 TUI randomizer
    function tui() {
        return random([
            "themes/tui/Amber/Main.qml",
            "themes/tui/Indigo/Main.qml",
            "themes/tui/Emerald/Main.qml",
            "themes/tui/Crimson/Main.qml",
            "themes/tui/Amethyst/Main.qml"
        ])
    }

    property var pool: [
        // 🔴 NieR → 25% (5/20)
        "themes/nier-automata/Main.qml",
        "themes/nier-automata/Main.qml",
        "themes/nier-automata/Main.qml",
        "themes/nier-automata/Main.qml",
        "themes/nier-automata/Main.qml",

        // ⚔️ Sword + Enfield → 25% (5/20)
        "themes/sword/Main.qml",
        "themes/sword/Main.qml",
        "themes/enfield/Main.qml",
        "themes/enfield/Main.qml",
        "themes/enfield/Main.qml",

        // 🌿 Remaining → 40% (8/20)
        pixelCore(), pixelCore(), pixelCore(), pixelCore(),
        "themes/forest/Main.qml",
        "themes/star-rail/Main.qml",
        pixelCore(),
        pixelAlt(),   // 👈 rare pixel variant

        // 🧠 TUI → 10% (2/20)
        tui(), tui()
    ]

    source: random(pool)
}
