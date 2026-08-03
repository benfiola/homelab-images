// Small hand-rolled syntax highlighter for the settings editor. No
// dependencies, no build step: a transparent <textarea> sits on top of a
// <pre><code> layer that this script keeps in sync on every keystroke and
// scroll event.
(function () {
  function escapeHtml(s) {
    return s
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  }

  function highlightLine(line) {
    var trimmed = line.replace(/\r$/, "");

    if (/^\s*[;#]/.test(trimmed)) {
      return '<span class="tok-comment">' + escapeHtml(trimmed) + "</span>";
    }

    var section = trimmed.match(/^(\[[^\]]*\])(.*)$/);
    if (section) {
      return (
        '<span class="tok-section">' + escapeHtml(section[1]) + "</span>" +
        escapeHtml(section[2])
      );
    }

    var kv = trimmed.match(/^([A-Za-z_][A-Za-z0-9_]*)=(.*?)(,?)$/);
    if (kv) {
      var key = kv[1];
      var value = kv[2];
      var trailingComma = kv[3];
      return (
        '<span class="tok-key">' + escapeHtml(key) + "</span>=" +
        '<span class="tok-val">' + escapeHtml(value) + "</span>" +
        (trailingComma ? '<span class="tok-punct">,</span>' : "")
      );
    }

    return escapeHtml(trimmed);
  }

  function highlight(text) {
    return text.split("\n").map(highlightLine).join("\n");
  }

  function setup(textareaId, highlightId) {
    var textarea = document.getElementById(textareaId);
    var code = document.getElementById(highlightId);
    if (!textarea || !code) {
      return;
    }

    function update() {
      code.innerHTML = highlight(textarea.value) + "\n";
    }

    function syncScroll() {
      var pre = code.parentElement;
      pre.scrollTop = textarea.scrollTop;
      pre.scrollLeft = textarea.scrollLeft;
    }

    textarea.addEventListener("input", update);
    textarea.addEventListener("scroll", syncScroll);
    update();
  }

  document.addEventListener("DOMContentLoaded", function () {
    setup("editable", "editable-highlight");
  });
})();
