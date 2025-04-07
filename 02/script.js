function generateSSML() {
    const voiceA = document.getElementById("voiceA").value;
    const voiceB = document.getElementById("voiceB").value;
    const conversation = document.getElementById("conversation").value;
    const lines = conversation.split("\n").filter((line) => line.trim() !== "");

    let ssml = `<speak xml:lang="vi-VN">\n`;

    lines.forEach((line) => {
        if (line.startsWith("A:")) {
            ssml += `  <voice name="${voiceA}">${line.slice(2).trim()}</voice>\n`;
        } else if (line.startsWith("B:")) {
            ssml += `  <voice name="${voiceB}">${line.slice(2).trim()}</voice>\n`;
        }
    });

    ssml += `</speak>`;

    let highlighted = ssml
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;");

    // Highlight tag names (e.g., speak, voice)
    highlighted = highlighted.replace(
        /(&lt;\/?)(\w+)/g,
        function (_, prefix, tag) {
            return prefix + '<span class="tag-name">' + tag + "</span>";
        }
    );

    // Highlight attributes and values (xml:lang, name)
    highlighted = highlighted.replace(
        /(\s)(xml:lang|name)=(")([^"]*)(")/g,
        function (_, space, attr, q1, val, q2) {
            return (
                space +
                '<span class="attr-name">' +
                attr +
                "</span>=" +
                q1 +
                '<span class="attr-value">' +
                val +
                "</span>" +
                q2
            );
        }
    );

    // Finally, highlight brackets (&lt;, &gt;)
    highlighted = highlighted
        .replace(/&lt;/g, '<span class="bracket">&lt;</span>')
        .replace(/&gt;/g, '<span class="bracket">&gt;</span>');

    const outputElement = document.getElementById("ssmlOutput");
    outputElement.innerHTML = highlighted;
    outputElement.dataset.ssml = ssml;
}

document.getElementById("copyBtn").onclick = function () {
    const data = document.getElementById("ssmlOutput").dataset.ssml;
    const btn = this;

    navigator.clipboard
        .writeText(data)
        .then(() => {
            btn.textContent = "✅ Đã copy!";
            setTimeout(() => {
                btn.textContent = "Copy";
            }, 2000); // đổi lại sau 2 giây
        })
        .catch((err) => {
            console.error("Failed to copy: ", err);
        });
};
