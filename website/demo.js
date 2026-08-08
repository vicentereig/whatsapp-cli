/* whatsapp-cli · live demo terminal
 * Scripted replay — not a live WhatsApp connection. See the caption under the
 * terminal for what each tab actually represents.
 */

(() => {
  "use strict";

  const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  /* ---------- Transcript data ---------- */
  /* whatsapp-cli's own documented `sync` progress output (README § Command: sync). */
  const SYNC_LINES = [
    "$ whatsapp-cli sync",
    "\u{1F680} Starting WhatsApp sync...",
    "✓ Connected to WhatsApp",
    "\u{1F504} Listening for messages... (Press Ctrl+C to stop)",
    "\u{1F4DC} Processing history sync (42 conversations)...",
    "\u{1F4AC} Synced 1234 messages...",
    "^C",
    "✓ Sync completed. Total messages synced: 1234",
  ];

  /* Illustrative companion script over the local `messages` table — sample
   * data, not a real chat backup, and not a whatsapp-cli subcommand. */
  const AGENT_LINES = [
    "$ python3 chat_backup_agent.py",
    "» which group chat grew the most this year?",
    "  SELECT chat_jid, COUNT(*) FROM messages",
    "  WHERE strftime('%Y', timestamp) = '2026'",
    "  GROUP BY chat_jid ORDER BY 2 DESC LIMIT 1;",
    "  Book Club — 1,842 messages (+340% vs. 2025)",
    "",
    "» when did I last hear from Mom?",
    "  SELECT MAX(timestamp) FROM messages",
    "  WHERE chat_jid = ? AND is_from_me = 0;",
    '  3 days ago, 21:14 — "sounds good, see you sunday"',
    "",
    "» any media I haven't backed up locally?",
    "  SELECT COUNT(*) FROM messages",
    "  WHERE media_type != '' AND local_path IS NULL;",
    "  212 messages reference media not yet downloaded.",
    "  Run `whatsapp-cli media download` to fetch them.",
  ];

  const TRANSCRIPTS = { sync: SYNC_LINES, agent: AGENT_LINES };
  const typed = { sync: false, agent: false };

  /* ---------- Typewriter ---------- */

  function typeInto(codeEl, lines, done) {
    const caret = document.createElement("span");
    caret.className = "terminal__caret";
    caret.setAttribute("aria-hidden", "true");
    codeEl.textContent = "";
    codeEl.appendChild(caret);

    if (reducedMotion) {
      codeEl.textContent = lines.join("\n");
      codeEl.appendChild(caret);
      if (done) done();
      return;
    }

    let line = 0;
    let char = 0;

    function step() {
      if (line >= lines.length) {
        if (done) done();
        return;
      }
      const current = lines[line];
      if (char <= current.length) {
        const soFar = lines.slice(0, line).concat(current.slice(0, char)).join("\n");
        codeEl.textContent = soFar;
        codeEl.appendChild(caret);
        char++;
        setTimeout(step, current.length === 0 ? 40 : 14);
      } else {
        line++;
        char = 0;
        setTimeout(step, 90);
      }
    }
    step();
  }

  function ensureTyped(name) {
    if (typed[name]) return;
    typed[name] = true;
    const panel = document.querySelector(`[data-panel="${name}"]`);
    const codeEl = panel.querySelector("code");
    typeInto(codeEl, TRANSCRIPTS[name]);
  }

  /* ---------- Tabs (ARIA tabs pattern, crossfade content) ---------- */

  const tabs = Array.from(document.querySelectorAll(".terminal__tab"));
  const panels = Array.from(document.querySelectorAll(".terminal__panel"));

  function activate(name) {
    tabs.forEach((tab) => {
      const isActive = tab.dataset.tab === name;
      tab.classList.toggle("is-active", isActive);
      tab.setAttribute("aria-selected", String(isActive));
      tab.tabIndex = isActive ? 0 : -1;
    });
    panels.forEach((panel) => {
      const isActive = panel.dataset.panel === name;
      panel.classList.toggle("is-active", isActive);
      panel.hidden = !isActive;
    });
    ensureTyped(name);
  }

  tabs.forEach((tab, i) => {
    tab.addEventListener("click", () => activate(tab.dataset.tab));
    tab.addEventListener("keydown", (e) => {
      if (e.key !== "ArrowRight" && e.key !== "ArrowLeft") return;
      e.preventDefault();
      const next = e.key === "ArrowRight"
        ? tabs[(i + 1) % tabs.length]
        : tabs[(i - 1 + tabs.length) % tabs.length];
      next.focus();
      activate(next.dataset.tab);
    });
  });

  /* Start the first tab's transcript once the terminal scrolls into view. */
  const terminal = document.querySelector("[data-terminal]");
  if (terminal) {
    const io = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          ensureTyped("sync");
          io.disconnect();
        }
      });
    }, { threshold: 0.3 });
    io.observe(terminal);
  }

  /* ---------- Copy to clipboard ---------- */

  document.querySelectorAll(".copy-btn").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const targetId = btn.dataset.copyTarget;
      const target = document.getElementById(targetId);
      if (!target) return;
      try {
        await navigator.clipboard.writeText(target.textContent.trim());
      } catch {
        /* Clipboard API unavailable (e.g. insecure context) — fail silently, label doesn't change. */
        return;
      }
      const label = btn.querySelector(".copy-btn__label");
      const original = label.textContent;
      btn.dataset.state = "copied";
      label.textContent = "Copied";
      setTimeout(() => {
        delete btn.dataset.state;
        label.textContent = original;
      }, 2500);
    });
  });

  /* ---------- Sticky CTA reveal ---------- */

  const stickyCta = document.querySelector("[data-sticky-cta]");
  const revealTrigger = document.querySelector("#commands");
  if (stickyCta && revealTrigger) {
    const io = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        stickyCta.classList.toggle("is-visible", entry.boundingClientRect.top < 0 || entry.isIntersecting);
      });
    }, { threshold: 0 });
    io.observe(revealTrigger);
  }
})();
