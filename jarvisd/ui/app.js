"use strict";
// Веб-настройки Джарвиса на колонке. Без библиотек; CSP запрещает inline-стили и обработчики.

const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const plural = (n, one, few, many) => { const a = Math.abs(n) % 100, b = a % 10; return a > 10 && a < 20 ? many : b === 1 ? one : b >= 2 && b <= 4 ? few : many; };
const tip = (t) => `<span class="tip" tabindex="0" data-tip="${esc(t)}">?</span>`;

// Иконки (lucide, 16px, линия 1.5) и кольцо — знак Джарвиса: повторяет световое кольцо колонки.
const ICONS = {
  play: `<polygon points="6 3 20 12 6 21 6 3"/>`,
  pause: `<rect x="6" y="4" width="4" height="16" rx="1"/><rect x="14" y="4" width="4" height="16" rx="1"/>`,
  skip: `<polygon points="5 4 15 12 5 20 5 4"/><rect x="17" y="5" width="2.5" height="14" rx="1"/>`,
  link: `<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>`,
  speaker: `<path d="M11 4.702a.705.705 0 0 0-1.203-.498L6.413 7.587A1.4 1.4 0 0 1 5.416 8H3a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h2.416a1.4 1.4 0 0 1 .997.413l3.383 3.384A.705.705 0 0 0 11 19.298z"/><path d="M16 9a5 5 0 0 1 0 6"/>`,
  x: `<path d="M18 6 6 18M6 6l12 12"/>`,
  right: `<path d="m9 18 6-6-6-6"/>`,
  done: `<circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/>`,
  todo: `<circle cx="12" cy="12" r="10"/>`,
  mic: `<path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2M12 19v3"/>`,
  stop: `<rect width="12" height="12" x="6" y="6" rx="2"/>`,
  tune: `<path d="M20 7h-9M14 17H5"/><circle cx="17" cy="17" r="3"/><circle cx="7" cy="7" r="3"/>`,
  read: `<path d="M18 6 7 17l-5-5"/><path d="m22 10-7.5 7.5L13 16"/>`,
  sent: `<path d="M20 6 9 17l-5-5"/>`,
  send: `<path d="m5 12 7-7 7 7M12 19V5"/>`,
  voice: `<path d="M11 4.702a.705.705 0 0 0-1.203-.498L6.413 7.587A1.4 1.4 0 0 1 5.416 8H3a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h2.416a1.4 1.4 0 0 1 .997.413l3.383 3.384A.705.705 0 0 0 11 19.298z"/><path d="M16 9a5 5 0 0 1 0 6M19.364 18.364a9 9 0 0 0 0-12.728"/>`,
  prev: `<polygon points="19 20 9 12 19 4 19 20"/><rect x="4.5" y="5" width="2.5" height="14" rx="1"/>`,
  music: `<path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>`,
  radio: `<path d="M4.9 19.1C1 15.2 1 8.8 4.9 4.9M7.8 16.2c-2.3-2.3-2.3-6.1 0-8.5M16.2 7.8c2.3 2.3 2.3 6.1 0 8.5M19.1 4.9C23 8.8 23 15.1 19.1 19"/><circle cx="12" cy="12" r="2"/>`,
  plus: `<path d="M5 12h14M12 5v14"/>`,
  ext: `<path d="M15 3h6v6M10 14 21 3M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>`,
  minus: `<path d="M5 12h14"/>`,
  shield: `<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"/>`,
  cloud: `<path d="M17.5 19H8a5 5 0 1 1 1.6-9.7A6 6 0 0 1 21 11.5 3.75 3.75 0 0 1 17.5 19Z"/>`,
  wifi: `<path d="M5 12.5a10 10 0 0 1 14 0M8.5 16a5 5 0 0 1 7 0M12 19.5h.01M1.5 9a15 15 0 0 1 21 0"/>`,
  search: `<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>`,
  cloud: `<path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/>`,
  moon: `<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/>`,
  mute: `<path d="M11 4.702a.705.705 0 0 0-1.203-.498L6.413 7.587A1.4 1.4 0 0 1 5.416 8H3a1 1 0 0 0-1 1v6a1 1 0 0 0 1 1h2.416a1.4 1.4 0 0 1 .997.413l3.383 3.384A.705.705 0 0 0 11 19.298z"/><path d="m22 9-6 6M16 9l6 6"/>`,
};
const icon = (name, cls = "") => `<svg class="ic${cls ? " " + cls : ""}" viewBox="0 0 24 24" aria-hidden="true">${ICONS[name]}</svg>`;
const ring = (state, cls = "") => `<svg class="ring ${state}${cls ? " " + cls : ""}" viewBox="0 0 48 48" aria-hidden="true"><circle class="track" cx="24" cy="24" r="20"/><circle class="arc" cx="24" cy="24" r="20"/></svg>`;
// Раздел настроек: слева название и пояснение, справа сами поля.
const sec = (title, desc, body, id = "", after = "") => `<section class="sec"><div class="sec-h"><h2${id ? ` id="${id}"` : ""}>${title}</h2>${desc ? `<p>${desc}</p>` : ""}${after}</div><div class="sec-b">${body}</div></section>`;

let S = null;          // состояние с колонки
let journal = [];
let page = "home";
let dirty = false;     // несохранённые правки на странице
let pollTimer = null;

async function api(method, url, body) {
  const r = await fetch(url, { method, headers: { "Content-Type": "application/json", "X-Jarvis": "1" }, body: body === undefined ? undefined : JSON.stringify(body), credentials: "same-origin" });
  let d = {};
  try { d = await r.json(); } catch { /* пусто */ }
  if (r.status === 401 || r.status === 428) { showGate(d.setup); throw new Error(d.error || "Нужно войти"); }
  if (!r.ok) throw new Error(d.error || `Ошибка ${r.status}`);
  return d;
}

function toast(msg, bad = false) {
  const t = $("#toast");
  t.textContent = msg;
  t.className = "toast show" + (bad ? " bad" : "");
  clearTimeout(toast.t);
  toast.t = setTimeout(() => (t.className = "toast"), bad ? 4500 : 2200);
}

async function act(btn, fn, okMsg) {
  if (btn) btn.disabled = true;
  try { await fn(); if (okMsg) toast(okMsg); } catch (e) { toast(e.message, true); } finally { if (btn) btn.disabled = false; }
}

// ---------------------------------------------------------------- вход
let gateSetup = false;
function showGate(setup) {
  gateSetup = !!setup;
  $("#app").classList.add("hidden");
  $("#gate").classList.remove("hidden");
  $("#gate-text").textContent = setup ? "Придумайте пароль для настроек колонки. Он понадобится, чтобы зайти сюда снова." : "Пароль от настроек колонки";
  $("#gate-pass2").classList.toggle("hidden", !setup);
  $("#gate-forgot").classList.toggle("hidden", setup);
  $("#gate-btn").textContent = setup ? "Сохранить пароль" : "Войти";
  $("#gate-pass").autocomplete = setup ? "new-password" : "current-password";
  $("#gate-pass").focus();
  clearInterval(pollTimer);
}

$("#gate-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const p = $("#gate-pass").value;
  $("#gate-err").textContent = "";
  try {
    if (gateSetup) {
      if (p !== $("#gate-pass2").value) throw new Error("Пароли не совпадают");
      await api("POST", "/api/setup", { password: p });
    } else {
      await api("POST", "/api/login", { password: p });
    }
    $("#gate-pass").value = $("#gate-pass2").value = "";
    $("#gate").classList.add("hidden");
    await start();
  } catch (err) { $("#gate").classList.remove("hidden"); $("#gate-err").textContent = err.message; }
});

$("#logout").addEventListener("click", async () => { await api("POST", "/api/logout").catch(() => {}); showGate(false); });
// меню на телефоне: шторка слева с затемнением; закрывается тапом мимо, Escape и переходом на страницу
const scrim = document.createElement("div");
scrim.className = "scrim";
document.body.append(scrim);
const setMenu = (open) => { $("#side").classList.toggle("open", open); document.body.classList.toggle("menu-open", open); };
$("#menu").addEventListener("click", () => setMenu(!$("#side").classList.contains("open")));
scrim.addEventListener("click", () => setMenu(false));
document.addEventListener("keydown", (e) => { if (e.key === "Escape" && $("#side").classList.contains("open")) setMenu(false); });

// ---------------------------------------------------------------- данные
async function reload() {
  S = await api("GET", "/api/state");
  S.live_voices ||= []; S.prompts ||= []; S.profiles ||= []; S.security ||= [];
  S.gemini_models ||= []; S.gemini_voice_groups ||= []; S.chatgpt_voice_groups ||= []; S.grok_voice_groups ||= []; S.chatgpt ||= {}; S.sys ||= {};
  renderSide();
}

async function start() {
  await reload();
  $("#app").classList.remove("hidden");
  $("#ver").textContent = "v" + S.version;
  route();
  clearInterval(pollTimer);
  pollTimer = setInterval(poll, 2000);
  poll();
}

async function poll() {
  let d;
  try { d = await api("GET", "/api/journal"); } catch { return; }
  if (d.version && S.version && d.version !== S.version) { location.reload(); return; }   // колонку обновили — берём новую панель
  journal = d.items || [];
  S.speaker.mic = d.mic; S.speaker.talking = d.talking; S.speaker.phase = d.phase || ""; S.sys = d.sys || {};
  renderStatus();
  if (d.volume >= 0) S.speaker.volume = d.volume;
  if (page === "home") { homeData(); drawVolume(); if (Date.now() - ST.at > 15000) statsData(); if (Date.now() - HL.at > 5000) homeLogs(); }
  if (page === "player") drawVolume();
  if (page === "charts" && Date.now() - CH.at > 15000) chartsData(false);
  if (page === "logs" && LOGS.auto) logsData(false);
  musicData();
  clearTimeout(poll.fast);
  if (S.speaker.talking && page === "home") poll.fast = setTimeout(poll, 500);
}


// Что сейчас делает колонка: вид кольца, заголовок и пояснение.
function speakerState() {
  const sp = S.speaker;
  if (sp.talking) return { ring: "talk", title: "Идёт разговор", text: "Говорите — пока разговор открыт, звать по имени не нужно" };
  if (!sp.mic) return { ring: "off", title: "Микрофон не подключён", text: "Колонка не присылает звук. Проверьте, что она включена и установка завершилась." };
  if (sp.wake_model && S.settings.wake_enabled) return { ring: "idle", title: "Жду слово «Джарвис»", text: "Скажите «Джарвис, …» или нажмите «Поговорить»" };
  return { ring: "mute", title: "Жду кнопку «Поговорить»", text: S.settings.wake_enabled ? "Модель слова «Джарвис» ещё не установлена" : "Отклик на слово «Джарвис» выключен" };
}

// Кольцо перерисовываем, только когда состояние сменилось, — иначе анимация дёргается на каждом опросе.
function setRing(box, state, cls) {
  if (!box || box.dataset.state === state) return;
  box.dataset.state = state;
  box.innerHTML = ring(state, cls);
}

const ORB_TEXT = { idle: "Колонка на связи — ждёт «Джарвис»", listening: "Слушает", thinking: "Думает", speaking: "Говорит", off: "Нет звука с колонки", mute: "На связи, но слово «Джарвис» выключено или нет модели" };
function renderStatus() {
  const st = speakerState(), ph = S.speaker.phase;
  const o = st.ring === "off" ? "off" : S.speaker.talking ? (ph || "listening") : st.ring === "mute" ? "mute" : "idle";
  const orb = $("#orb");
  if (orb.dataset.s !== o) { orb.dataset.s = o; orb.setAttribute("aria-label", ORB_TEXT[o]); orb.title = ORB_TEXT[o]; }
}

// ---------------------------------------------------------------- сайдбар
const avatar = (p, big = false) => `<span class="av c${p.color || 1}${big ? " big" : ""}">${esc((p.name || "?").trim()[0] || "?")}</span>`;
const voiceName = (id) => (S.live_voices.find((v) => v.id === id)?.name) || id || "cedar";
const engineName = (p) => ({ gemini: "Gemini Live", chatgpt: "GPT-Live по подписке", grok: "Grok Voice" })[p.engine] || "GPT-Live";

function renderSide() {
  $("#upd-dot")?.classList.toggle("hidden", !S.update_available);
  $("#profiles").innerHTML = S.profiles.map((p) => `<div class="prof${page === "p:" + p.id ? " on" : ""}" draggable="true" data-id="${esc(p.id)}">
    ${avatar(p)}<div class="meta"><div class="n">${esc(p.name)}</div><div class="s">${engineName(p)} · ${esc(voiceName(p.voice))}</div></div>
    ${p.id === S.active ? `<span class="act" title="Сейчас отвечает"></span>` : ""}</div>`).join("");
  $$("#profiles .prof").forEach((el) => {
    el.addEventListener("click", () => go("#p/" + el.dataset.id));
    el.addEventListener("dragstart", (e) => { el.classList.add("drag"); e.dataTransfer.setData("text/plain", el.dataset.id); });
    el.addEventListener("dragend", () => el.classList.remove("drag"));
    el.addEventListener("dragover", (e) => e.preventDefault());
    el.addEventListener("drop", async (e) => {
      e.preventDefault();
      const from = e.dataTransfer.getData("text/plain"), to = el.dataset.id;
      if (from === to) return;
      const ids = S.profiles.map((p) => p.id).filter((id) => id !== from);
      ids.splice(ids.indexOf(to), 0, from);
      await act(null, async () => { await api("POST", "/api/profiles/order", { ids }); await reload(); });
    });
  });
  $("#nav-train")?.classList.toggle("hidden", !S.settings.dev_mode);
  $$(".nav a").forEach((a) => a.classList.toggle("on", a.dataset.page === page));
  navPill();
}

$("#add-profile").addEventListener("click", () => go("#p/new"));

// ---------------------------------------------------------------- маршруты
function go(hash) { location.hash = hash; }
window.addEventListener("hashchange", () => {
  if (!S) return;   // ещё не вошли — страницы рисовать не из чего
  if (dirty && !confirm("Есть несохранённые изменения. Уйти без сохранения?")) { history.back(); return; }
  route();
});
window.addEventListener("beforeunload", (e) => { if (dirty) e.preventDefault(); });

function route() {
  dirty = false;
  $("#side").classList.remove("open");
  document.body.classList.remove("menu-open");
  const h = location.hash.slice(1) || "home";
  const v0 = $("#view"); v0.oninput = v0.onchange = v0.onclick = null;   // обработчики прошлой страницы
  if (h.startsWith("p/")) { page = "p:" + h.slice(2); viewProfile(h.slice(2)); }
  else {
    page = ["home", "player", "charts", "logs", "characters", "connections", "speaker", "security", "training"].includes(h) ? h : "home";
    if (page === "training" && !S.settings.dev_mode) page = "security";
    ({ home: viewHome, player: viewPlayer, charts: viewCharts, logs: viewLogs, characters: viewCharacters, connections: viewConnections, speaker: viewSpeaker, training: viewTraining, security: viewSecurity })[page]();
  }
  renderSide();
  window.scrollTo(0, 0);
  const v = $("#view");          // одно появление на переход между страницами
  v.classList.remove("enter"); void v.offsetWidth; v.classList.add("enter");
}

// ---------------------------------------------------------------- главная
function setupSteps() {
  return [
    { done: !!S.keys.openai || !!S.keys.gemini || !!S.keys.grok || !!S.chatgpt.signed_in, title: "Ключ OpenAI, Gemini, Grok или вход в ChatGPT", text: "Без него Джарвис не говорит и не думает", href: "#connections" },
    { done: S.vless.set && !S.vless.error, title: "VPN (ссылка VLESS)", text: "OpenAI не работает напрямую из России и Беларуси", href: "#connections" },
    { done: !!S.keys.yandex || S.settings.home_via !== "jarvis", title: "Умный дом", text: "Через Алису работает сразу; для режима «Джарвис» нужен токен Яндекса", href: "#speaker" },
    { done: !!S.settings.city, title: "Ваш город", text: "Для погоды и новостей без уточнений", href: "#speaker" },
  ];
}

function viewHome() {
  const p = S.profiles.find((x) => x.id === S.active) || S.profiles[0];
  const steps = setupSteps(), left = steps.filter((x) => !x.done), doneN = steps.length - left.length;
  $("#view").innerHTML = `<header class="ph ph-row"><div><h1>Главная</h1><p class="sub"><span class="dot" id="home-dot"></span><span id="home-sub"></span></p></div></header>
  <div class="kpis stagger" id="kpis">${KPI.map(kpiHtml).join("")}</div>
  <div class="home">
  <section class="tg">
    <header class="tg-h"><span class="tg-av"><span id="now-ring" class="ringbox"></span>${avatar(p)}</span>
      <div class="tg-who"><b>${esc(p.name)}</b><span id="tg-status" class="tg-st"></span></div>
      <a class="ib lg" href="#p/${esc(p.id)}" aria-label="Настроить ассистента">${icon("tune")}</a></header>
    <div class="tg-body" id="feed"></div>
    <form class="tg-f" id="say-form"><input type="text" id="say" maxlength="600" placeholder="Сообщение" autocomplete="off" aria-label="Сообщение Джарвису">
      <button class="ib lg" id="say-voice" type="button" data-tip="Читать ответ вслух на колонке" aria-label="Читать ответ вслух"></button>
      <button class="rb hidden" id="stop" type="button" data-tip="Оборвать разговор" aria-label="Стоп">${icon("stop", "solid")}</button>
      <button class="rb primary hidden" id="send" type="submit" aria-label="Отправить">${icon("send")}</button>
      <button class="rb primary" id="talk" type="button" data-tip="Колонка сразу начнёт слушать — как будто вы сказали «Джарвис»" aria-label="Поговорить">${icon("mic")}</button></form>
  </section>
  <aside class="rail">
    ${left.length ? `<div class="card"><h2>Осталось настроить <span class="count">${doneN} из ${steps.length}</span></h2>
      <progress class="bar" value="${doneN}" max="${steps.length}"></progress>
      ${steps.map((x) => `<a class="step${x.done ? " done" : ""}" href="${x.href}">${icon(x.done ? "done" : "todo")}<span class="sb"><b>${esc(x.title)}</b><small>${esc(x.text)}</small></span>${x.done ? "" : icon("right")}</a>`).join("")}</div>` : ""}
    <div class="card"><h2>Состояние</h2><dl class="stat">
      <div><dt>Микрофон ${tip("Звук с микрофонов колонки после шумо- и эхоподавления Яндекса.")}</dt><dd id="t-mic"></dd></div>
      <div><dt>Слово «Джарвис» ${tip("Распознаётся прямо на колонке маленькой нейросетью — без интернета и сразу.")}</dt><dd id="t-wake"></dd></div>
      <div><dt>Интернет ${tip("Как колонка ходит в OpenAI: через ваш VPN (VLESS) или напрямую.")}</dt><dd id="t-net"></dd></div>
      <div><dt>Свободно на диске ${tip("На флешке колонки мало места. Если свободно меньше 5 МБ, обновления могут не встать.")}</dt><dd id="t-disk"></dd></div>
    </dl></div>
    <section class="card github-support" id="github-support" aria-label="Поддержать проект">
      <button class="github-close" id="github-dismiss" type="button" aria-label="Больше не показывать просьбу поддержать проект" title="Больше не показывать">${icon("x")}</button>
      <span class="github-mark" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M9 19c-4.3 1.3-4.3-2.2-6-2.6M15 22v-3.9a3.4 3.4 0 0 0-.9-2.6c3-.3 6.2-1.5 6.2-6.9a5.4 5.4 0 0 0-1.5-3.7 5 5 0 0 0-.1-3.7S17.5.8 15 2.6a12.8 12.8 0 0 0-6 0C6.5.8 5.3 1.2 5.3 1.2a5 5 0 0 0-.1 3.7 5.4 5.4 0 0 0-1.5 3.7c0 5.4 3.2 6.6 6.2 6.9a3.4 3.4 0 0 0-.9 2.6V22"/></svg></span>
      <div class="github-copy"><h2>Поддержать YaJavis</h2><p>Нравится проект? Оставьте звезду на GitHub.</p>
        <a class="github-link" href="https://github.com/newfpv/YaJavis" target="_blank" rel="noopener noreferrer">Поставить звезду ${icon("ext")}</a>
      </div>
    </section>
    ${playerHtml()}
    <section class="card home-logs tight" id="hlogs"><div class="card-head"><h2>Логи</h2><a class="meta hl-all" href="#logs">Все логи ${icon("right")}</a></div><div class="logbox" id="hlogbox"></div></section>
  </aside>
  </div>`;
  feedHtml = ""; feedSeen = 0;
  ST.at = 0; drawKpis(); statsData();
  playerSetup(); drawVolume(); if (M) drawPlayer(); musicData();
  homeLogsSetup();
  try { if (localStorage.getItem("yajavis.star.dismissed") === "1") $("#github-support")?.remove(); } catch {}
  $("#github-dismiss")?.addEventListener("click", () => {
    $("#github-support")?.remove();
    try { localStorage.setItem("yajavis.star.dismissed", "1"); } catch {}
  });
  $("#talk").addEventListener("click", (e) => act(e.currentTarget, () => api("POST", "/api/talk"), "Слушаю…"));
  $("#stop").addEventListener("click", (e) => act(e.currentTarget, () => api("POST", "/api/stop")));
  // написать текстом: пока в поле что-то есть, вместо микрофона — кнопка отправки
  const say = $("#say"), voiceBtn = $("#say-voice");
  const pref = { get: () => { try { return localStorage.getItem("jarvis.speak") !== "0"; } catch { return true; } }, set: (v) => { try { localStorage.setItem("jarvis.speak", v ? "1" : "0"); } catch { /* не страшно */ } } };
  let speak = pref.get();
  const drawVoice = () => { voiceBtn.innerHTML = icon(speak ? "voice" : "mute"); voiceBtn.classList.toggle("on", speak); voiceBtn.dataset.tip = speak ? "Ответ прозвучит на колонке. Нажмите, чтобы отвечал только текстом" : "Ответ только текстом. Нажмите, чтобы читал вслух на колонке"; };
  drawVoice();
  voiceBtn.addEventListener("click", () => { speak = !speak; pref.set(speak); drawVoice(); });
  const swap = () => { const typing = say.value.trim() !== ""; $("#send").classList.toggle("hidden", !typing); $("#talk").classList.toggle("hidden", typing); };
  say.addEventListener("input", swap);
  $("#say-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    const text = say.value.trim();
    if (!text) return;
    say.value = ""; swap();
    try { await api("POST", "/api/say", { text, speak }); } catch (err) { toast(err.message, true); say.value = text; swap(); return; }
    $("#feed").scrollTop = $("#feed").scrollHeight;
    poll(); setTimeout(poll, 700);
  });
  homeData();
}

// Служебные события разговора — в переписке идут строкой по центру, как в мессенджерах.
const SERVICE = { "умный дом (быстро)": "Выполнил сам", "умный дом → Алиса": "Передал Алисе", "стоп": "Остановлен", "ложное срабатывание": "Показалось, что позвали" };
const dayLabel = (d) => {
  const day = (x) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const diff = Math.round((day(new Date()) - day(d)) / 864e5);
  return diff === 0 ? "Сегодня" : diff === 1 ? "Вчера" : d.toLocaleDateString("ru-RU", { day: "numeric", month: "long" });
};
let feedHtml = "", feedSeen = 0;   // что уже нарисовано: не трогаем переписку, пока в ней ничего не изменилось

function serviceText(it) {
  const base = SERVICE[it.route];
  if (it.route === "умный дом → Алиса" && it.answer) return it.answer.replace(/^передал/, "Передал");
  return it.answer && it.answer !== "Готово." ? base + " · " + it.answer : base;
}

function homeData() {
  const feed = $("#feed");
  if (!feed) return;
  const st = speakerState();
  setRing($("#now-ring"), st.ring, "av");
  const status = $("#tg-status");
  status.textContent = S.speaker.talking ? "говорит и слушает…" : st.title.replace(/^Жду/, "ждёт").replace(/^Микрофон/, "микрофон");
  status.className = "tg-st" + (S.speaker.talking ? " live" : st.ring === "off" ? " warn" : "");
  $("#say").placeholder = S.speaker.talking ? "Идёт разговор — можно и написать" : "Сообщение";
  const sys = S.sys || {};
  drawKpis();
  const up = sys.uptime_s || 0, d = Math.floor(up / 86400), h = Math.floor((up % 86400) / 3600), m = Math.floor((up % 3600) / 60);
  const hd = "dot " + (S.speaker.mic ? "on" : "off");
  if ($("#home-dot").className !== hd) $("#home-dot").className = hd;
  $("#home-sub").textContent = `${S.speaker.model || "Яндекс Станция"} · Джарвис ${S.version}${up ? ` · колонка работает ${d ? `${d} д ${h} ч` : h ? `${h} ч ${m} мин` : `${m} мин`}` : ""}`;
  const disk = $("#t-disk");
  disk.textContent = sys.disk_total_mb ? `${sys.disk_free_mb} из ${sys.disk_total_mb} МБ` : "—";
  disk.className = "v " + (sys.disk_total_mb && sys.disk_free_mb < 5 ? "warn" : "ok");
  $("#stop").classList.toggle("hidden", !S.speaker.talking);
  const set = (id, text, cls) => { const el = $(id); el.textContent = text; el.className = "v " + cls; };
  set("#t-mic", S.speaker.mic ? "на связи" : "нет звука", S.speaker.mic ? "ok" : "bad");
  set("#t-wake", !S.settings.wake_enabled ? "выключено" : S.speaker.wake_model ? "слушаю" : "нет модели", !S.settings.wake_enabled ? "warn" : S.speaker.wake_model ? "ok" : "warn");
  set("#t-net", S.vless.error ? "ошибка VPN" : S.vless.set ? "через VPN" : "напрямую", S.vless.error ? "bad" : S.vless.set ? "ok" : "warn");

  const items = journal.filter((it) => it.route !== "ложное срабатывание" || it.text).slice(0, 40).reverse();   // старые сверху
  const time = (ts) => new Date(ts * 1000).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
  const build = (rich) => {
  let day = "", html = "";
  for (const it of items) {
    const d = dayLabel(new Date(it.ts * 1000)), fresh = it.ts > feedSeen ? " new" : "";
    if (d !== day) { day = d; html += `<div class="svc day">${d}</div>`; }
    const service = SERVICE[it.route], tm = time(it.ts);
    const answered = !!(it.answer || it.error || service);
    if (it.text) html += `<div class="b me${fresh}"><span class="t">${esc(it.text)}</span><span class="m">${tm}${icon(answered ? "read" : "sent")}</span></div>`;
    if (service) html += `<div class="svc${fresh}">${esc(serviceText(it))}</div>`;
    else if (it.answer) html += `<div class="b him${fresh}"${rich ? ` data-k="${it.ts}"` : ""}><span class="t">${rich ? answerHtml(it.ts, it.answer, feedHtml !== "") : esc(it.answer)}</span><span class="m">${typeof it.audio_s === "number" && it.audio_s > 0 ? `ответ через ${it.audio_s.toFixed(1)} с · ` : ""}${tm}</span></div>`;
    if (it.error) html += `<div class="b him err${fresh}"><span class="t">${esc(it.error)}</span><span class="m">${tm}</span></div>`;
    if (it.live && !answered) {
      const ph = S.speaker.phase;
      if (!it.text) html += `<div class="b me live${fresh}" aria-label="Говорите"><span class="wave">${"<i></i>".repeat(7)}</span><span class="t">Слушаю</span></div>`;
      else if (ph === "speaking") html += `<div class="b him speak${fresh}" aria-label="Джарвис отвечает"><span class="eqw">${"<i></i>".repeat(5)}</span><span class="t">Отвечает</span></div>`;
      else html += `<div class="b him think"${rich ? ` data-k="${it.ts}"` : ""} aria-label="Джарвис думает"><span class="think-t">Думаю</span></div>`;
    }
  }
  return html;
  };
  let html = build(false);
  if (!items.length) html = `<div class="tg-empty"><b>Здесь будет переписка</b><span>Скажите «Джарвис, какая погода?» — вопрос и ответ появятся тут</span></div>`;
  if (html === feedHtml) return;
  const first = feedHtml === "", pinned = feed.scrollHeight - feed.scrollTop - feed.clientHeight < 80;
  const plain = html;
  if (items.length) html = build(true);
  feed.innerHTML = html; feedHtml = plain;
  if (!first) streamBubbles(feed); else for (const b of feed.querySelectorAll(".b.him[data-k]")) bubbleSize[b.dataset.k] = { w: b.offsetWidth, h: b.offsetHeight };
  if (items.length) feedSeen = Math.max(feedSeen, ...items.map((it) => it.ts));
  if (first || pinned) feed.scrollTop = feed.scrollHeight;   // новое сообщение показываем, если человек не листал вверх
}

// ---------------------------------------------------------------- ассистент
function viewProfile(id) {
  const isNew = id === "new";
  const base = S.profiles.find((x) => x.id === id);
  if (!isNew && !base) { go("#home"); return; }
  const p = structuredClone(base || { id: "", name: "", color: (S.profiles.length % 6) + 1, mode: "live", voice: "cedar", backend_model: S.backend_models[0], effort: "low", priority: true, web_search: true, idle_s: 8, prompt: S.prompts[0]?.id || "" });
  p.led ||= { ...(S.settings.led || {}) }; p.led.brightness ||= 45;
  p.volume ||= S.settings.volume || 40; p.city ??= S.settings.city || ""; p.timezone ||= S.settings.timezone || "UTC";
  p.engine ||= "openai";
  if(p.mode==="classic")p.engine="elevenlabs";
  p.llm_model ||= p.backend_model;
  p.tts_model ||= "eleven_flash_v2_5";
  if(p.engine==="elevenlabs")p.backend_model=p.llm_model;
  const active = p.id === S.active;
  $("#view").innerHTML = `<header class="ph who"><span id="pav" class="ringbox">${avatar(p, true)}</span><div><h1>${isNew ? "Новый ассистент" : esc(p.name)}</h1><p class="sub">${isNew ? "Свой голос, модель и характер" : active ? "Сейчас отвечает на «Джарвис»" : "Не активен"}</p></div></header>
  ${sec("Основное", "Как ассистент выглядит в списке и кем он себя считает",
    `<div class="grid g2"><div class="f"><label>Имя ${tip("Как ассистент называется в списке слева. На слово активации не влияет.")}</label><input type="text" id="p-name" maxlength="40" value="${esc(p.name)}" placeholder="Например, Джарвис"></div>
      <div class="f"><label>Цвет</label><div class="colors" id="p-color">${[1, 2, 3, 4, 5, 6].map((c) => `<button class="av c${c}${p.color === c ? " on" : ""}" data-c="${c}" type="button" aria-label="Цвет ${c}"></button>`).join("")}</div></div></div>
    <div class="f"><label>Характер ${tip("Системный промпт: кто такой Джарвис и как он общается. Характеры редактируются в разделе «Характеры».")}</label><select id="p-prompt">${S.prompts.map((c) => `<option value="${esc(c.id)}"${c.id === p.prompt ? " selected" : ""}>${esc(c.name)}</option>`).join("")}</select></div>`)}
  ${sec("Мозг", "Выберите живой разговор или классическую цепочку: распознавание речи → текстовая модель → озвучка",
    `<div class="choice" id="p-engine">
      <button class="opt" data-engine="elevenlabs" type="button"><b>ElevenLabs — классический</b><small>Распознавание Scribe и голос ElevenLabs. Текстовая модель OpenAI думает и вызывает инструменты. Нужны ключи ElevenLabs и OpenAI</small></button>
      <button class="opt" data-engine="eleven_realtime" type="button"><b>ElevenLabs — живой разговор</b><small>Eleven Realtime Mini или Max: распознавание, мозг, голос и перебивания вместе. Нужен только ключ ElevenLabs</small></button>
      <button class="opt" data-engine="chatgpt" type="button"><b>GPT-Live по подписке</b><small>Ваш ChatGPT Plus или Pro: минуты не оплачиваются, есть лимиты подписки. Нужен вход в «Подключениях»</small></button>
      <button class="opt" data-engine="openai" type="button"><b>GPT-Live</b><small>OpenAI по ключу API. Около 5 центов за минуту разговора</small></button>
      <button class="opt" data-engine="gemini" type="button"><b>Gemini Live</b><small>Google. Примерно вдвое дешевле, есть бесплатный уровень. Нужен ключ Gemini в «Подключениях»</small></button>
      <button class="opt" data-engine="grok" type="button"><b>Grok Voice</b><small>xAI. Около 5 центов за минуту, сам ищет в интернете. Нужен ключ xAI в «Подключениях»</small></button>
    </div>
      <div class="eng" data-for="elevenlabs"><div class="f half"><label>Распознавание речи</label><select id="p-stt-model">${["scribe_v3_live","scribe_v2_realtime"].map(m=>`<option value="${m}"${m===(p.stt_model || "scribe_v2_realtime")?" selected":""}>${m==="scribe_v3_live"?"Scribe v3 Live — определяет конец мысли":"Scribe v2 Realtime"}</option>`).join("")}</select></div><div class="f half"><label>Модель озвучки</label><select id="p-tts-model">${["eleven_v4_turbo","eleven_v4","eleven_v3_conversational","eleven_v3","eleven_flash_v2_5","eleven_multilingual_v2"].map(m=>`<option value="${m}"${m===p.tts_model?" selected":""}>${m}</option>`).join("")}</select></div></div>
      <div class="eng" data-for="eleven_realtime"><div class="f half"><label>Разговорная модель</label><select id="p-eleven-rtmodel">${["eleven_realtime_v1_mini","eleven_realtime_v1_max"].map(m=>`<option value="${m}"${m===p.live_model?" selected":""}>${m.endsWith("max")?"Realtime v1 Max — сложные задачи и инструменты":"Realtime v1 Mini — быстрый разговор"}</option>`).join("")}</select></div></div>
      <div class="eng" data-for="elevenlabs eleven_realtime"><p class="note">Голос выбирается ниже из вашего аккаунта ElevenLabs.</p><button class="btn" id="p-el-refresh" type="button">Обновить голоса ElevenLabs</button></div>
      <div class="eng" data-for="gemini"><div class="f half"><label>Модель ${tip("С бесплатным ключом из AI Studio все эти модели работают без оплаты, но у каждой свой дневной лимит. Когда он кончается, Джарвис сам переходит на следующую модель в списке.")}</label><select id="p-gmodel">${(S.gemini_model_info?.length ? S.gemini_model_info : S.gemini_models.map((id) => ({ id, name: id, note: "" }))).map((m) => `<option value="${esc(m.id)}"${m.id === p.live_model ? " selected" : ""}>${esc(m.name)}${m.note ? ` — ${esc(m.note)}` : ""}</option>`).join("")}</select></div></div>
    <div class="eng" data-for="openai chatgpt elevenlabs"><div class="f half"><label>Текстовая модель ${tip("Чем умнее модель, тем дольше она думает. Для голосового помощника обычно хватает самой быстрой.")}</label><select id="p-model">${S.backend_models.map((m) => `<option${m === p.backend_model ? " selected" : ""}>${esc(m)}</option>`).join("")}</select></div>
      <div class="f"><label>Сколько думать ${tip("Доступные уровни зависят от выбранной текстовой модели: не все модели поддерживают «не думать». Немного — быстрее, средне и долго — для сложных вопросов.")}</label><div class="seg" id="p-effort">${S.efforts.map((e) => `<button type="button" data-v="${e}" class="${e === p.effort ? "on" : ""}">${({ none: "не думать", low: "немного", medium: "средне", high: "долго" })[e] || e}</button>`).join("")}</div></div>
    <label class="sw"><input type="checkbox" id="p-priority"${p.priority ? " checked" : ""}><span class="tr"></span>Приоритет ${tip("Запросы обрабатываются в быстрой очереди OpenAI: ответ быстрее, но стоит дороже.")}</label></div>
    <div class="stack">
      <label class="sw"><input type="checkbox" id="p-web"${p.web_search ? " checked" : ""}><span class="tr"></span>Поиск в интернете ${tip("Для погоды, новостей, курсов и всего свежего. Без него Джарвис знает только то, чему его научили.")}</label></div>
    <div class="f narrow"><label>Конец разговора, с ${tip("Если вы оба молчите столько секунд, разговор закрывается и колонка снова ждёт «Джарвис». Пока разговор открыт, звать по имени не нужно.")}</label><input type="number" id="p-idle" min="2" max="60" step="1" value="${p.idle_s}"></div>`)}
  ${sec("Голос", "Кнопка рядом с голосом — колонка скажет им пробную фразу. Первый раз это около цента, дальше фраза играет с самой колонки бесплатно",
    `<div id="voices" class="vgroups"></div>
    <details id="v-custom"><summary>Создать свой голос по описанию</summary><div class="inner">
      <div class="f"><label>Название</label><input type="text" id="v-name" placeholder="Например, Бархатный"></div>
      <div class="f"><label>Описание ${tip("Тембр, возраст, манера, темп, эмоции. Подражать конкретным людям и персонажам OpenAI не даёт, но описать манеру можно. Создание бесплатное.")}</label><textarea id="v-prompt" placeholder="Низкий спокойный мужской голос, лет сорока, говорит неторопливо, с лёгкой иронией…"></textarea></div>
      <div><button class="btn" id="v-create" type="button">Создать голос</button></div></div></details>
    <div class="f"><label>Громкость голоса ${tip("Насколько голос Джарвиса громче или тише остального звука колонки. Общая громкость колонки — на главной: голос меняется вместе с ней, но мягче — на тихой колонке его всё равно слышно, а на громкой он не орёт громче Алисы. Голосом: «Джарвис, громче/тише».")}<b id="p-vol-v" class="val">${p.volume}%</b></label><input type="range" id="p-vol" min="10" max="100" step="5" value="${p.volume}"></div>`)}
  ${sec("Подсветка", "Цвет кольца у этого ассистента. Нажмите цвет — колонка покажет его на 3 секунды",
    `<div id="p-led">${ledPickerHtml(p.led, "p")}</div>
    <div class="f"><label>Яркость <b id="p-br-v" class="val">${p.led.brightness}%</b></label><input type="range" id="p-br" min="5" max="100" step="5" value="${p.led.brightness}"></div>
    ${S.settings.led?.enabled === false ? `<p class="note">Сейчас подсветка выключена для всех — включается на странице «Колонка».</p>` : ""}`)}
  <div class="savebar"><button class="btn primary" id="p-save">${isNew ? "Создать" : "Сохранить"}</button>
    ${!isNew && !active ? `<button class="btn" id="p-activate" data-tip="Этот ассистент будет отвечать на «Джарвис»">Сделать активным</button>` : ""}
    ${!isNew && !active ? `<button class="btn ghost danger" id="p-del">Удалить</button>` : ""}<span class="dirty hidden" id="dirty">есть несохранённые изменения</span></div>`;

  const mark = () => { dirty = true; $("#dirty").classList.remove("hidden"); };
  const effortOptions = (model) => S.model_efforts?.[model] || S.efforts;
  const refreshEffortOptions = () => {
    const allowed = effortOptions($("#p-model").value);
    if (!allowed.includes(p.effort)) { p.effort = "low"; mark(); }
    $$("#p-effort button").forEach((b) => {
      b.style.display = allowed.includes(b.dataset.v) ? "" : "none";
      b.classList.toggle("on", b.dataset.v === p.effort);
    });
  };
  refreshEffortOptions();
  $("#p-model").addEventListener("change", () => { p.backend_model = $("#p-model").value; refreshEffortOptions(); mark(); });
  $("#view").oninput = mark;
  $("#view").onchange = mark;
  $("#p-name").addEventListener("input", (e) => { p.name = e.target.value; $("#pav").innerHTML = avatar(p, true); });
  $("#p-color").addEventListener("click", (e) => { const b = e.target.closest("button"); if (!b) return; p.color = +b.dataset.c; $$("#p-color button").forEach((x) => x.classList.toggle("on", x === b)); $("#pav").innerHTML = avatar(p, true); mark(); });
  $("#p-vol").addEventListener("input", (e) => ($("#p-vol-v").textContent = e.target.value + "%"));
  ledPickerBind($("#p-led"), p.led, "p", () => mark());
  $("#p-br").addEventListener("input", (e) => { p.led.brightness = +e.target.value; $("#p-br-v").textContent = e.target.value + "%"; paintLeds($("#p-led"), p.led, "p"); });
  $("#p-br").addEventListener("change", () => { mark(); if (S.speaker.leds) api("POST", "/api/leds/preview", { mode: "listening", color: p.led.listening, brightness: p.led.brightness }).catch(() => {}); });
  $("#p-effort").addEventListener("click", (e) => { const b = e.target.closest("button"); if (!b) return; p.effort = b.dataset.v; $$("#p-effort button").forEach((x) => x.classList.toggle("on", x === b)); mark(); });

  let elevenVoicesList=[],elevenLoaded=false,elevenLoading=false,elevenVoiceError="";
  const loadElevenVoices=async()=>{
    if(elevenLoaded||elevenLoading)return;elevenLoading=true;
    try{const r=await api("GET","/api/elevenlabs/voices");elevenVoicesList=r.voices || [];elevenVoiceError="";}
    catch(err){elevenVoiceError=err.message;}finally{elevenLoading=false;elevenLoaded=true;if(["elevenlabs","eleven_realtime"].includes(p.engine)&&$("#voices"))drawVoices();}
  };
  const drawVoices = () => {
    if(["elevenlabs","eleven_realtime"].includes(p.engine)){
      $("#voices").innerHTML=elevenVoiceError?`<p class="err">${esc(elevenVoiceError)}</p>`:!elevenLoaded?`<p class="note">Загружаю голоса вашего аккаунта ElevenLabs…</p>`:!elevenVoicesList.length?`<p class="note">В аккаунте нет доступных голосов.</p>`:
        `<div class="voices">${elevenVoicesList.map(v=>`<div class="voice${v.voice_id===p.tts_voice?" on":""}" data-v="${esc(v.voice_id)}"><div class="vn"><b>${esc(v.name)}</b><small>${esc([v.category,v.labels?.language].filter(Boolean).join(" · "))}</small></div><button class="ib" type="button" data-play="${esc(v.voice_id)}" title="Послушать на колонке" aria-label="Послушать голос">${icon("play","solid")}</button></div>`).join("")}</div>`;
      return;
    }
    const gem = p.engine !== "openai";
    const groups = [...(({ gemini: S.gemini_voice_groups, chatgpt: S.chatgpt_voice_groups, grok: S.grok_voice_groups })[p.engine] || S.voice_groups)];
    if (!gem && S.live_voices.length) groups.unshift({ name: "Мои голоса", voices: S.live_voices.map((v) => ({ ID: v.id, Note: v.prompt, mine: true, name: v.name })) });
    $("#voices").innerHTML = groups.map((g) => `<div class="vgroup"><div class="gh">${esc(g.name)}</div><div class="voices">${g.voices.map((v) => `
      <div class="voice${v.ID === p.voice ? " on" : ""}" data-v="${esc(v.ID)}"><div class="vn"><b>${esc(v.name || v.ID)}</b><small>${esc(v.Note)}</small></div>
      <button class="ib" type="button" data-play="${esc(v.ID)}" title="Послушать на колонке" aria-label="Послушать на колонке">${icon("play", "solid")}</button>${v.mine ? `<button class="ib" type="button" data-delv="${esc(v.ID)}" title="Удалить голос" aria-label="Удалить голос">${icon("x")}</button>` : ""}</div>`).join("")}</div></div>`).join("");
  };
  // движок: свои модели и свои голоса
  const drawEngine = () => {
    $$("#p-engine .opt").forEach((b) => b.classList.toggle("on", b.dataset.engine === p.engine));
    $$(".eng").forEach((el) => el.classList.toggle("hidden", !el.dataset.for.split(" ").includes(p.engine)));
    $("#v-custom").classList.toggle("hidden", p.engine !== "openai");
    if(["elevenlabs","eleven_realtime"].includes(p.engine)){drawVoices();loadElevenVoices();return;}
    const known = (({ gemini: S.gemini_voice_groups, chatgpt: S.chatgpt_voice_groups, grok: S.grok_voice_groups })[p.engine] || S.voice_groups).flatMap((g) => g.voices.map((v) => v.ID.toLowerCase()));
    const mine = S.live_voices.some((v) => v.id === p.voice);
    if (!known.includes(String(p.voice).toLowerCase()) && !(p.engine === "openai" && mine)) p.voice = ({ gemini: "Charon", chatgpt: "cove", grok: "rex" })[p.engine] || "cedar";
    drawVoices();
  };
  drawEngine();
  $("#p-el-refresh").addEventListener("click",()=>{elevenLoaded=false;elevenVoiceError="";drawVoices();loadElevenVoices();});
  $("#p-engine").addEventListener("click", (e) => { const b = e.target.closest(".opt"); if (!b || b.dataset.engine === p.engine) return; p.engine = b.dataset.engine; drawEngine(); mark(); });
  $("#voices").addEventListener("click", async (e) => {
    const play = e.target.closest("[data-play]"), del = e.target.closest("[data-delv]"), card = e.target.closest(".voice");
    if (play) { e.stopPropagation(); await act(play, async () => { const r = await api("POST", "/api/preview", { voice: play.dataset.play, engine: p.engine, tts_model:$("#p-tts-model").value }); toast(r.cached ? "Играет на колонке" : "Прозвучало на колонке и сохранено"); }); return; }
    if (del) { e.stopPropagation(); if (!confirm("Удалить этот голос из списка?")) return; await act(del, async () => { await api("DELETE", "/api/voices?id=" + encodeURIComponent(del.dataset.delv)); await reload(); drawVoices(); }); return; }
    if (card) { if(["elevenlabs","eleven_realtime"].includes(p.engine))p.tts_voice=card.dataset.v;else p.voice = card.dataset.v; drawVoices(); mark(); }
  });
  $("#v-create").addEventListener("click", (e) => act(e.target, async () => {
    const r = await api("POST", "/api/voices", { name: $("#v-name").value.trim(), prompt: $("#v-prompt").value.trim() });
    await reload(); p.voice = r.id; drawVoices(); mark();
  }, "Голос создан — послушайте его кнопкой рядом"));

  $("#p-save").addEventListener("click", (e) => act(e.target, async () => {
    Object.assign(p, { name: $("#p-name").value.trim(), prompt: $("#p-prompt").value, backend_model: $("#p-model").value,
      priority: $("#p-priority").checked, web_search: $("#p-web").checked, idle_s: +$("#p-idle").value || 8, mode: "live",
      volume: +$("#p-vol").value,
      live_model: p.engine === "gemini" ? $("#p-gmodel").value : "" });
    p.mode=p.engine==="elevenlabs"?"classic":"live";
    if(p.engine==="eleven_realtime"){p.live_model=$("#p-eleven-rtmodel").value;if(!p.tts_voice)throw new Error("Выберите голос ElevenLabs");}
    if(p.engine==="elevenlabs"){p.llm_model=p.backend_model;p.tts_model=$("#p-tts-model").value;p.stt_engine="elevenlabs";p.stt_model=$("#p-stt-model").value;if(!p.tts_voice)throw new Error("Выберите голос ElevenLabs");}
    const r = await api("PUT", "/api/profile", p);
    dirty = false; await reload();
    if (isNew) go("#p/" + r.id); else route();
  }, "Сохранено"));
  $("#p-activate")?.addEventListener("click", (e) => act(e.target, async () => { await api("POST", "/api/active", { id: p.id }); await reload(); route(); }, `Теперь отвечает «${p.name}»`));
  $("#p-del")?.addEventListener("click", (e) => { if (!confirm(`Удалить ассистента «${p.name}»?`)) return; act(e.target, async () => { await api("DELETE", "/api/profile?id=" + encodeURIComponent(p.id)); dirty = false; await reload(); go("#home"); }, "Удалено"); });
}

// ---------------------------------------------------------------- характеры
// ---------------------------------------------------------------- подключения
// Подключения — плитками, как «Характеры»: на плитке назначение и состояние, всё остальное — в окне по клику.
// Новое подключение = ещё одна запись в connDefs().
const CONN = { open: "", net: {}, haEntities: [] };

function haRoomRow(r = {}) {
  const options = (filter, value) => `<option value="">Выберите…</option>` +
    [...new Map([...(value ? [{entity_id:value,attributes:{friendly_name:value}}] : []), ...CONN.haEntities.filter(filter)].map(e => [e.entity_id,e])).values()]
      .map(e => `<option value="${esc(e.entity_id)}"${e.entity_id===value?" selected":""}>${esc(e.attributes?.friendly_name || e.entity_id)}</option>`).join("");
  return `<div class="card" data-ha-room data-ha-binding="${esc(JSON.stringify(r))}"><div class="f"><label>Название комнаты</label><input class="ha-name" value="${esc(r.name || "")}" placeholder="Кухня"></div>
    <div class="f"><label>Пылесос</label><select class="ha-vacuum">${options(e=>e.entity_id.startsWith("vacuum."),r.vacuum)}</select></div>
    <div class="f"><label>Сценарий или действие уборки этой комнаты</label><select class="ha-entity">${options(e=>/^(script|notify|text)\./.test(e.entity_id),r.entity)}</select></div>
    <div class="f"><label>Номера зон на карте — только для действия Xiaomi</label><input class="ha-ids" value="${esc(r.room_ids || "")}" placeholder="Например, 3 или 3,9"></div>
    <button class="btn ghost" data-ha-remove>Убрать комнату</button></div>`;
}
const NET_NAMES = { openai: "OpenAI", gemini: "Gemini", grok: "Grok", elevenlabs: "ElevenLabs", youtube: "YouTube", telegram: "Telegram" };
const VPN_NAMES = [["chatgpt", "Подписка ChatGPT"], ["openai", "OpenAI"], ["gemini", "Google Gemini"], ["grok", "xAI Grok"],
  ["elevenlabs", "ElevenLabs"], ["telegram", "Telegram"], ["youtube", "YouTube"], ["smartlife", "Smart Life"], ["xiaomi", "Xiaomi Home"], ["gdrive", "Google Диск"], ["yandex", "Яндекс"]];

function telegramConn() {
  const tg = S.telegram || {};
  const isSet = !!tg.set;
  const isRunning = !!tg.running;
  const isEnabled = !!tg.enabled;
  const stText = !isSet ? "не настроен" : !isEnabled ? "выключен" : tg.bot_name ? `@${tg.bot_name}` : "работает";

  return {
    id: "telegram",
    group: "services",
    mono: "TG",
    name: "Telegram-бот",
    about: "Управление колонкой из Telegram, отправка заметок голосом и удалённый чат",
    more: "Позволяет давать команды колонке (/say, /vol, /status) и общаться с Джарвисом из любого места",
    on: isSet && isEnabled,
    need: false,
    state: stText,
    body: `<p class="note">Создайте бота в Telegram и укажите его токен и ваш Chat ID. Бот принимает команды только от вас и добавленных пользователей.</p>
      <div class="f">
        <label for="tg-token">Токен бота из @BotFather ${tg.token ? `<small class="muted">(${esc(tg.token)})</small>` : ""}</label>
        <input type="password" id="tg-token" placeholder="${isSet ? "Оставьте пустым, чтобы не менять" : "Например, 1234567890:AAH..."}" autocomplete="off">
      </div>
      <div class="f">
        <label for="tg-chat-id">Telegram Chat ID (можно несколько через запятую)</label>
        <input type="text" id="tg-chat-id" value="${esc(tg.chat_id || "")}" placeholder="Например: 134460301, 987654321" autocomplete="off">
        <small class="muted" style="margin-top:-6px;display:block;">Укажите ID через запятую (например, ваш и жены), чтобы ботом могли пользоваться члены семьи.</small>
      </div>
      <label class="sw">
        <input type="checkbox" id="tg-enabled"${isEnabled ? " checked" : ""}>
        <span class="tr"></span>Включить бота
      </label>
      <div class="row">
        <button class="btn primary" id="tg-save">Сохранить</button>
        <button class="btn" id="tg-test">Проверить связь</button>
      </div>
      ${vpnRow("telegram")}
      <details>
        <summary>Инструкция: как создать бота и узнать Chat ID</summary>
        <ol class="how">
          <li>Откройте в Telegram бота <b>@BotFather</b> и отправьте команду <code>/newbot</code>.</li>
          <li>Придумайте название и юзернейм (например, <code>MyMidiJarvis_bot</code>).</li>
          <li>Скопируйте выданный API токен и вставьте в поле выше.</li>
          <li>Найдите своего созданного бота в Telegram и нажмите <b>Start</b>.</li>
          <li><b>Как узнать Chat ID:</b> откройте бота <b>@userinfobot</b> — он покажет ваш цифровой Id. Либо просто напишите любому новому боту — он сам пришлёт ваш Chat ID в ответ.</li>
          <li><b>Как добавить жену или семью:</b> пусть супруга напишет вашему боту любое сообщение — бот пришлёт её Chat ID. Скопируйте и допишите его в поле Chat ID через запятую (например: <code>134460301, 987654321</code>) и нажмите «Сохранить».</li>
        </ol>
      </details>
      <details>
        <summary>Команды бота и голосовые фразы</summary>
        <ul class="how">
          <li><code>/say &lt;текст&gt;</code> — сказать фразу вслух на Станции</li>
          <li><code>/vol 0-10</code> — изменить громкость колонки</li>
          <li><code>/status</code> — посмотреть состояние колонки и сервисов</li>
          <li><code>/remember &lt;факт&gt;</code> — сохранить факт в долговременную память</li>
          <li><code>/memory</code> — список всего, что помнит Джарвис</li>
          <li><code>/forget &lt;слово&gt;</code> — удалить факт из памяти</li>
          <li><b>Любой текст</b> — диалог с Джарвисом с учётом вашей памяти</li>
          <li><b>Голосом на колонке</b>: <i>«Джарвис, отправь в телеграм список покупок: ...»</i></li>
        </ul>
      </details>`,
    drop: isSet ? `<button class="btn ghost danger" id="tg-clear">Убрать бота</button>` : ""
  };
}

function memoryConn() {
  const mem = S.memory || {};
  const items = mem.items || [];
  const n = items.length;
  const word = n % 10 === 1 && n % 100 !== 11 ? "факт" : [2, 3, 4].includes(n % 10) && ![12, 13, 14].includes(n % 100) ? "факта" : "фактов";

  const list = items.length ? `
    <div class="ek-list">
      ${items.map((it) => `
        <div class="ek on">
          <b>🧠</b>
          <span class="ek-st" title="${esc(it.fact)}">${esc(it.fact)}</span>
          <span class="spacer"></span>
          <button class="ib" data-mem-del="${esc(it.id)}" aria-label="Удалить" data-tip="Удалить факт">${icon("x")}</button>
        </div>
      `).join("")}
    </div>`
    : `<p class="note">Память пока пуста. Добавьте факты ниже или просто скажите колонке: «Джарвис, запомни, что...»</p>`;

  return {
    id: "memory",
    group: "services",
    mono: "🧠",
    name: "Личная память",
    about: "Долговременные воспоминания о вас, ваших привычках, имени и семье",
    more: "Факты автоматически учитываются моделью при каждом разговоре",
    on: n > 0,
    state: n > 0 ? `${n} ${word}` : "пусто",
    body: `<p class="note">Джарвис помнит эти факты между разговорами и перезапусками колонки, учитывая их в ответах.</p>
      ${list}
      <div class="f">
        <label for="mem-new">Новый факт</label>
        <div class="row">
          <input type="text" id="mem-new" placeholder="Например: Хозяина зовут Александр, пьёт кофе без сахара" autocomplete="off">
          <button class="btn primary" id="mem-add">Запомнить</button>
        </div>
      </div>
      <p class="note">💡 <b>Голосовое управление:</b> скажите колонке: <i>«Джарвис, запомни: я люблю кофе без сахара»</i> или <i>«Джарвис, забудь про кофе»</i>.</p>`,
    drop: ""
  };
}
function smartLifeConn() {
  const t = S.smartlife || {}, lg = t.login || {}, devs = t.devices || [];
  const local = devs.filter((d) => d.local).length;
  const list = devs.length ? `<div class="xm-devs">${devs.map((d) => `<div class="xm-dev sl-dev${d.online ? "" : " off"}"><span class="sl-way ${d.local ? "lan" : "cloud"}" data-tip="${d.local ? "По домашней сети — мгновенно" : "Через облако Tuya — около полсекунды. Устройства Zigbee (через шлюз) всегда идут так"}">${icon(d.local ? "wifi" : "cloud")}</span><b>${esc(d.name)}</b><small>${esc([d.room, d.product].filter(Boolean).join(" · "))}${d.online ? "" : " · не в сети"}</small></div>`).join("")}</div>
      <p class="note">${local ? `По домашней сети, мгновенно: ${local} из ${devs.length}. Остальные — через облако Tuya.` : "Устройства в домашней сети ещё не откликнулись — пока команды идут через облако Tuya. Обычно они находятся за минуту."}</p>`
    : `<p class="note">Устройств не нашлось. Если они есть в Smart Life — нажмите «Обновить устройства».</p>`;
  return { id: "smartlife", group: "home", mono: "SL", name: "Smart Life", about: "Лампы, розетки и другие устройства Tuya — напрямую, без Алисы",
    more: "«Включи/выключи …» выполняется сразу по домашней сети, за доли секунды",
    on: !!t.signed_in, wait: !t.signed_in && !!lg.code, state: t.signed_in ? `${devs.length} ${plural(devs.length, "устройство", "устройства", "устройств")}` : lg.code ? "ждёт QR-код" : "не подключено",
    body: t.signed_in ? `<div class="cn-ok">${icon("done")}<span>Колонка вошла в Smart Life</span></div>${list}
        <div class="row"><button class="btn" id="sl-refresh">Обновить устройства</button></div>${vpnRow("smartlife")}`
      : lg.code ? `<div class="xm-qr"><img src="/api/smartlife/qr?t=${lg.at || 0}" alt="QR-код для входа в Smart Life" width="220" height="220">
          <ol class="how cn-steps"><li>Откройте Smart Life → значок сканера в правом верхнем углу.</li><li>Отсканируйте код и подтвердите вход.</li><li>Это окно само увидит вход и найдёт устройства.</li></ol></div>`
      : `${lg.error ? `<p class="err">${esc(lg.error)}</p>` : ""}
        <div class="f"><label for="sl-code">Код пользователя ${tip("В Smart Life: «Я» → шестерёнка «Настройки» → «Аккаунт и безопасность» → «Код пользователя». Это не пароль.")}</label>
          <div class="row"><input type="text" id="sl-code" placeholder="Например, AbC123x" autocomplete="off" spellcheck="false"><button class="btn primary" id="sl-in">Получить QR-код</button></div></div>
        <p class="note">Код пользователя — в Smart Life: «Я» → «Настройки» → «Аккаунт и безопасность». Потом отсканируете QR в самом приложении. Пароль колонка не видит.</p>${vpnRow("smartlife")}`,
    drop: t.signed_in ? `<button class="btn ghost danger" id="sl-out">Выйти</button>` : "" };
}
function xiaomiConn() {
  const x = S.xiaomi || {}, lg = x.login || {}, devs = x.devices || [];
  const n = devs.length, word = n % 10 === 1 && n % 100 !== 11 ? "устройство" : [2, 3, 4].includes(n % 10) && ![12, 13, 14].includes(n % 100) ? "устройства" : "устройств";
  const list = devs.length ? `<div class="xm-devs">${devs.map((d) => `<div class="xm-dev${d.online ? "" : " off"}"><b>${esc(d.name)}</b><small>${esc([d.room, d.type || d.model, d.region && "сервер " + d.region].filter(Boolean).join(" · "))}${d.online ? "" : " · не в сети"}</small></div>`).join("")}</div>`
    : `<p class="note">Устройств не нашлось. Если они есть в Mi Home — нажмите «Обновить устройства».</p>`;
  return { id: "xiaomi", group: "home", mono: "Mi", name: "Xiaomi Home", about: "Лампы, розетки, пылесосы и другие устройства из приложения Mi Home",
    more: "Джарвис управляет ими голосом: «включи лампу в спальне», «запусти пылесос»",
    on: !!x.signed_in, wait: !x.signed_in && !!lg.code, state: x.signed_in ? `${n} ${word}` : lg.code ? "ждёт QR-код" : "не подключено",
    body: x.signed_in ? `<div class="cn-ok">${icon("done")}<span>Колонка вошла в Xiaomi Home${x.region ? `<small>${x.region.includes(",") ? "серверы" : "сервер"} ${esc(x.region)}</small>` : ""}</span></div>${list}
        <div class="row"><button class="btn" id="xm-refresh">Обновить устройства</button></div>${vpnRow("xiaomi")}`
      : lg.code ? `<div class="xm-qr"><img src="/api/xiaomi/qr?t=${lg.at || 0}" alt="QR-код для входа в Xiaomi" width="220" height="220">
          <ol class="how cn-steps"><li>Отсканируйте код камерой телефона или в приложении Mi Home: <b>+</b> → <b>Сканировать</b>.</li>
          <li>Войдите в свой аккаунт Xiaomi и подтвердите вход.</li><li>Это окно само увидит вход и найдёт устройства.</li></ol></div>
          ${lg.url ? `<p class="note">Открыли эту страницу на телефоне? <a href="${esc(lg.url)}" target="_blank" rel="noopener">Войти по ссылке</a></p>` : ""}`
      : `${lg.error ? `<p class="err">${esc(lg.error)}</p>` : ""}<div class="row"><button class="btn primary" id="xm-in">Войти по QR-коду</button></div>
          <p class="note">Пароль колонка не видит: вы подтверждаете вход на телефоне. Устройства берутся из того же аккаунта, что и в Mi Home.</p>${vpnRow("xiaomi")}`,
    drop: x.signed_in ? `<button class="btn ghost danger" id="xm-out">Выйти</button>` : "" };
}
// галочка «через VPN» у подключения
function vpnRow(svc) {
  const on = !!S.vpn?.[svc], set = !!S.vless?.set;
  return `<div class="cn-vpn"><label class="sw"><input type="checkbox" data-vpn="${svc}"${on ? " checked" : ""}><span class="tr"></span>Через VPN</label>
    <small class="muted">${!set ? "VPN не задан — галочка заработает, когда добавите его в «Сеть → VPN»" : on ? "идёт через ваш VPN" : "напрямую, без VPN"}</small></div>`;
}
function geminiKeysHtml() {
  const list = S.gemini_keys || [];
  if (list.length < 2) return list.length && list[0].spent ? `<p class="note">Лимит кончился у ${list[0].spent} из ${list[0].models} моделей этого ключа. Добавьте ещё ключ — Джарвис перейдёт на него сам.</p>` : "";
  return `<div class="ek-list">${list.map((k, i) => `<div class="ek${!k.spent && !list.slice(0, i).some((x) => x.spent < x.models) ? " on" : ""}${k.spent >= k.models ? " rest" : ""}"><b>${i + 1}</b><code>${esc(k.tail)}</code>
      <span class="ek-st">${k.spent >= k.models ? "лимит кончился · ждёт" : k.spent ? `лимит кончился у ${k.spent} из ${k.models} моделей` : "лимит есть"}</span><span class="spacer"></span>
      <button class="ib" data-gk-remove="${i}" aria-label="Убрать ключ ${i + 1}">${icon("x")}</button></div>`).join("")}</div>
    <p class="note">У каждого ключа свой бесплатный лимит. Кончился у всех моделей первого — Джарвис сам переходит на следующий.</p>`;
}
function elevenKeysHtml() {
  const list = S.eleven_keys || [];
  if (!list.length) return "";
  const date = (u) => new Date(u * 1000).toLocaleDateString("ru-RU", { day: "numeric", month: "long" });
  return `<div class="ek-list">${list.map((k, i) => `<div class="ek${k.active ? " on" : ""}${k.until ? " rest" : ""}"><b>${i + 1}</b><code>${esc(k.tail)}</code>
      <span class="ek-st">${k.until ? `квота кончилась · ждёт до ${date(k.until)}` : k.active ? "сейчас работает" : "в запасе"}</span><span class="spacer"></span>
      ${k.until ? `<button class="btn sm ghost" data-ek-reset="${i}" data-tip="Например, если докупили символы">Вернуть</button>` : ""}
      <button class="ib" data-ek-remove="${i}" aria-label="Убрать ключ ${i + 1}">${icon("x")}</button></div>`).join("")}</div>
    <p class="note">Работает первый ключ с квотой. Кончилась — ключ откладывается до сброса лимита (обычно через месяц), и Джарвис сам берёт следующий.</p>`;
}

function connDefs() {
  const k = S.keys, v = S.vless, cg = S.chatgpt, lg = cg.login || {}, yt = S.youtube || {}, ya = yt.account || {};
  const keyBody = (id, place, how, label) => `<div class="f"><label for="k-${id}">${label ? label[0] : k[id] ? "Новый ключ" : "Ключ"}</label><div class="row"><input type="password" id="k-${id}" placeholder="${place}" autocomplete="off"><button class="btn primary" data-key="${id}">${label ? label[1] : k[id] ? "Заменить" : "Сохранить"}</button></div></div>
    ${how ? `<details${k[id] ? "" : " open"}><summary>Где его взять</summary><ol class="how">${how}</ol></details>` : ""}`;
  const key = (id, o) => ({ id, group: o.group, mono: o.mono, name: o.name, about: o.about, more: o.more,
    on: !!k[id], need: !k[id] && !o.optional, state: k[id] ? `ключ ${esc(String(k[id]).replace(/^задан,?\s*/, ""))}` : o.optional ? "не подключено" : "нужен ключ",
    body: (o.top || "") + keyBody(id, o.place, o.how, o.label) + (o.vpn ? vpnRow(o.vpn) : ""), drop: k[id] ? `<button class="btn ghost danger" data-clear="${id}">${o.dropText || "Убрать ключ"}</button>` : "" });
  return [
    { id:"homeassistant", group:"home", mono:"HA", name:"Home Assistant", about:"Устройства дома, Xiaomi Home и уборка выбранных комнат",
      on:!!S.home_assistant?.configured, state:S.home_assistant?.configured?"подключено":"не подключено",
      body:`<p class="note">Подключите свой сервер Home Assistant. Аккаунт Xiaomi, регион и устройства выбираются в самом Home Assistant.</p>
        <div class="f"><label for="ha-url">Адрес сервера</label><input id="ha-url" type="url" value="${esc(S.home_assistant?.url || "")}" placeholder="http://192.168.1.10:8123"></div>
        <div class="f"><label for="ha-token">Долгосрочный токен из профиля Home Assistant</label><input id="ha-token" type="password" autocomplete="off" placeholder="${S.home_assistant?.configured?"Оставьте пустым, чтобы сохранить текущий":"Вставьте токен"}"></div>
        <div class="row"><button class="btn primary" id="ha-save">Проверить и сохранить</button>${S.home_assistant?.configured?`<button class="btn" id="ha-load">Загрузить устройства</button><a href="${esc(S.home_assistant.url)}/config/integrations/dashboard" target="_blank" rel="noopener">Открыть Home Assistant</a>`:""}</div>
        <p class="note">Для Xiaomi добавьте интеграцию Xiaomi Home и войдите в свой аккаунт. Название комнаты в приложении и номер зоны на карте пылесоса могут различаться.</p>
        <h3>Уборка по комнатам</h3><p class="note">Выберите готовый сценарий уборки либо действие Xiaomi с номерами зон. Без соответствия Джарвис попросит настроить комнату.</p>
        <div id="ha-rooms">${(S.home_assistant?.rooms || []).map(haRoomRow).join("")}</div><div class="row"><button class="btn" id="ha-discover-rooms">Найти комнаты пылесоса</button><button class="btn" id="ha-add-room">Добавить комнату</button></div>`,
      drop:S.home_assistant?.configured?`<button class="btn ghost danger" id="ha-disconnect">Отключить</button>`:"" },
    { id: "chatgpt", group: "voice", mono: "GP", name: "Подписка ChatGPT", about: "Голос Джарвиса за счёт подписки Plus или Pro — без оплаты API за минуты",
      more: "Включается у ассистента: движок «GPT-Live по подписке».",
      on: !!cg.signed_in, state: cg.signed_in ? esc(["вход выполнен", cg.plan].filter(Boolean).join(" · ")) : lg.code ? "ждёт код" : "не подключено",
      wait: !cg.signed_in && !!lg.code,
      body: cg.signed_in ? `<div class="cn-ok">${icon("done")}<span>Колонка вошла в ChatGPT${cg.plan ? ` · <b>${esc(cg.plan)}</b>` : ""}${cg.email ? `<small>${esc(cg.email)}</small>` : ""}</span></div>
          ${cg.limited_until * 1000 > Date.now() ? `<p class="err">Кончился лимит голоса в подписке. До ${new Date(cg.limited_until * 1000).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" })} Джарвис говорит через Gemini${S.keys?.gemini ? "" : " — но ключа Gemini нет, добавьте его ниже"}.</p>` : ""}
          <p class="note">Один чат на все разговоры (память общая для всех ассистентов): ChatGPT помнит ${cg.chat?.turns ? `${cg.chat.turns} ${plural(cg.chat.turns, "последнюю реплику", "последние реплики", "последних реплик")}` : "прошлые реплики"}, даже после перезапуска колонки.</p>
          <div class="row"><button class="btn" id="cg-newchat"${cg.chat?.turns ? "" : " disabled"}>Начать новый чат</button></div>`
        : lg.code ? `<ol class="how cn-steps"><li>Откройте <a href="${esc(lg.url)}" target="_blank" rel="noopener">${esc(lg.url.replace("https://", ""))}</a> (из России и Беларуси — через VPN) и войдите в свой ChatGPT.</li><li>Введите код: <b class="code-big">${esc(lg.code)}</b></li><li>Это окно само увидит вход — ничего нажимать не нужно.</li></ol>`
        : `${lg.error ? `<p class="err">${esc(lg.error)}</p>` : ""}<div class="row"><button class="btn primary" id="cg-in">Войти в ChatGPT</button></div>
          <p class="note">Колонка получит свой вход, пароль она не видит. Способ неофициальный, как у Codex: OpenAI может его изменить.</p>`
        + vpnRow("chatgpt"),
      drop: cg.signed_in ? `<button class="btn ghost danger" id="cg-out">Выйти</button>` : "" },
    key("openai", { vpn: "openai", group: "voice", mono: "AI", name: "OpenAI", place: "sk-…", about: "Голос и «мозг» Джарвиса. Живой разговор — примерно 5 центов в минуту",
      how: "<li>Откройте platform.openai.com и войдите.</li><li>Слева — API keys → Create new secret key.</li><li>Скопируйте ключ (начинается с sk-) и вставьте сюда.</li><li>Пополните баланс в разделе Billing — от 5 $.</li>" }),
    key("gemini", { vpn: "gemini", top: geminiKeysHtml(), label: k.gemini ? ["Ещё ключ — в запас", "Добавить"] : null,
      dropText: (S.gemini_keys || []).length > 1 ? "Убрать все ключи" : "Убрать ключ", group: "voice", mono: "Ge", name: "Google Gemini", place: "AIza…", optional: true, about: "Gemini Live — примерно вдвое дешевле GPT-Live, есть бесплатный уровень. Можно добавить несколько ключей: у каждого свой бесплатный лимит",
      more: "Какой движок отвечает, выбирается у ассистента.",
      how: "<li>Откройте aistudio.google.com и войдите в аккаунт Google (из России и Беларуси — через VPN).</li><li>Нажмите Get API key → Create API key.</li><li>Скопируйте ключ и вставьте сюда.</li>" }),
    key("grok", { vpn: "grok", group: "voice", mono: "xA", name: "xAI Grok", place: "xai-…", optional: true, about: "Grok Voice — сам ищет в интернете",
      more: "Какой движок отвечает, выбирается у ассистента.",
      how: "<li>Откройте console.x.ai и войдите.</li><li>Пополните баланс (Billing → Credits) — без кредитов ключ не работает.</li><li>API Keys → Create API key, скопируйте ключ (начинается с xai-) и вставьте сюда.</li>" }),
    key("elevenlabs", { vpn: "elevenlabs", top: elevenKeysHtml(), label: k.elevenlabs ? ["Ещё ключ — в запас", "Добавить"] : null,
      dropText: (S.eleven_keys || []).length > 1 ? "Убрать все ключи" : "Убрать ключ", group: "voice", mono: "11", name: "ElevenLabs", place: "sk_…", optional: true, about: "Распознавание речи и голоса классического режима. Выбирается в настройках ассистента → Мозг" }),
    key("yandex", { vpn: "yandex", group: "home", mono: "Я", name: "Яндекс Умный дом", place: "y0_…", about: "Джарвис сам управляет светом, пылесосом и сценариями",
      more: "В режиме «через Алису» можно обойтись без него.",
      how: "<li>Нужен OAuth-токен Яндекса с правами iot:view и iot:control.</li><li>Его выдаёт приложение, зарегистрированное на oauth.yandex.ru (тип «Веб-сервисы», права «Умный дом»).</li><li>Вставьте токен (начинается с y0_).</li>" }),
    smartLifeConn(),
    xiaomiConn(),
    { id: "youtube", group: "home", mono: "YT", name: "YouTube", about: "Музыка и видео голосом. Вход — для ваших рекомендаций и видео 18+", more: "Там, где YouTube заблокирован (Россия), включите «Через VPN»",
      on: !!(ya.signed_in || yt.set), state: ya.signed_in ? "вход выполнен" : yt.set ? "куки загружены" : ya.code ? "ждёт код" : "не подключено",
      wait: !ya.signed_in && !!ya.code,
      body: ya.code ? `<div class="ycode"><span>Откройте <a href="${esc(ya.url)}" target="_blank" rel="noopener">${esc(ya.url.replace("https://", ""))}</a> и введите</span><b class="code-big" role="button" tabindex="0" title="Нажмите, чтобы скопировать">${esc(ya.code)}</b><small class="muted">окно само увидит вход</small></div>`
        : ya.signed_in ? `<div class="cn-ok">${icon("done")}<span>Колонка вошла в YouTube<small>видео 18+ тоже играют</small></span></div>`
        : `${ya.error ? `<p class="err">${esc(ya.error)}</p>` : ""}<div class="row"><button class="btn primary" id="ya-in">Войти в YouTube</button><small class="muted">кодом, как на телевизоре — пароль колонка не видит</small></div>
          <details class="mini"><summary>Не получается кодом? Куки браузера</summary><ol class="how"><li>Расширение «Get cookies.txt LOCALLY» → окно инкогнито → войдите на youtube.com.</li><li>Export (Netscape) → скопируйте весь текст сюда → закройте окно, не выходя из аккаунта.</li></ol>
          <textarea id="yt-cookies" class="code" placeholder="# Netscape HTTP Cookie File…" spellcheck="false" aria-label="Куки YouTube"></textarea>
          <div class="row"><button class="btn sm" id="yt-save">${yt.set ? "Заменить" : "Сохранить и проверить"}</button></div></details>`
        + vpnRow("youtube"),
      drop: ya.signed_in ? `<button class="btn ghost danger" id="ya-out">Выйти</button>` : yt.set ? `<button class="btn ghost danger" id="yt-clear">Убрать куки</button>` : "" },
    { id: "vpn", group: "net", icon: "shield", name: "VPN для Джарвиса", about: "Через VPN идёт только сам Джарвис, Алиса — всегда напрямую. Что именно пускать через VPN — галочки ниже",
      on: !!v.set && !v.error, bad: !!v.error, need: !v.set, state: v.set ? (v.error ? "ошибка" : esc(v.name || v.host)) : "не задан",
      body: `${v.set ? `<div class="vpn ${v.error ? "bad" : "ok"}"><b>${esc(v.name || v.host)}</b><small>${esc(v.host)}:${esc(v.port)} · ${esc(v.network)} + ${esc(v.security)}</small></div>` : `<p class="note">Пока не задан — колонка ходит в интернет напрямую. Вставьте ссылку vless://… из вашего VPN (3x-ui, Marzban, Hiddify и т. п.).</p>`}
        ${v.error ? `<p class="err">${esc(v.error)}</p>` : ""}
        <textarea id="vless" class="code" placeholder="vless://…" spellcheck="false" aria-label="Ссылка VLESS"></textarea>
        <div class="row"><button class="btn primary" id="vless-save">${v.set ? "Заменить ссылку" : "Сохранить ссылку"}</button><span class="spacer"></span><button class="btn" data-net>Проверить связь</button></div>
        <div class="netres" id="net-res"></div>
        <h3>Что идёт через VPN</h3><div class="vpn-list">${VPN_NAMES.map(([id, n]) => `<label class="sw"><input type="checkbox" data-vpn="${id}"${S.vpn?.[id] ? " checked" : ""}><span class="tr"></span>${n}</label>`).join("")}</div>`,
      drop: v.set ? `<button class="btn ghost danger" id="vless-clear">Убрать VPN</button>` : "" },
    telegramConn(),
    memoryConn(),
  ];
}

const CONN_GROUPS = [
  ["voice", "Голос и мозг", "Чем Джарвис говорит и думает. Достаточно одного"],
  ["services", "Telegram и память", "Управление через Telegram и долговременная память"],
  ["home", "Дом и музыка", ""],
  ["net", "Сеть", ""]
];

function viewConnections() {
  $("#view").innerHTML = `<header class="ph ph-row"><div><h1>Подключения</h1><p class="sub">Ключи хранятся только на колонке и обратно в браузер не отдаются</p></div>
    <div class="actions"><button class="btn" data-net data-tip="Колонка достучится до OpenAI, Gemini, Grok и ElevenLabs с вашими ключами и покажет время ответа">${icon("radio")}Проверить связь</button></div></header>
  <div id="conns"></div>
  <div class="modal hidden" id="cn-modal" role="dialog" aria-modal="true" aria-labelledby="cn-mt"><div class="modal-bg" data-close></div><div class="modal-card cn-card" id="cn-card"></div></div>`;
  CONN.open = "";
  const view = $("#view");
  view.oninput = view.onchange = null;
  view.onclick = connClick;
  view.onkeydown = (e) => {
    if (page !== "connections") return; // обработчик висит на #view и после ухода со страницы
    const t = e.target.closest(".cn[data-conn]");
    if (t && (e.key === "Enter" || e.key === " ")) { e.preventDefault(); connOpen(t.dataset.conn); }
    if (e.key === "Escape" && CONN.open) connClose();
    if (e.key === "Enter" && e.target.matches("input[id^='k-']")) $(`[data-key="${e.target.id.slice(2)}"]`)?.click();
    if (e.key === "Enter" && e.target.id === "sl-code") $("#sl-in")?.click();
    if (e.key === "Enter" && e.target.id === "mem-new") $("#mem-add")?.click();
    if (e.key === "Enter" && (e.target.id === "tg-token" || e.target.id === "tg-chat-id")) $("#tg-save")?.click();
  };
  drawConns();
}

function connTile(c) {
  const net = CONN.net[c.id];
  const cls = c.bad || (net && !net.ok) ? "bad" : c.on ? "ok" : c.need ? "warn" : "";
  const state = net ? (net.ok ? `${c.state} · ${net.ms} мс` : esc(net.error || "нет связи")) : c.state;
  return `<article class="card cn${c.on ? " on" : ""}" data-conn="${c.id}" tabindex="0" role="button" aria-label="${esc(c.name)}">
    <div class="cn-h"><span class="char-av cn-av">${c.icon ? icon(c.icon) : esc(c.mono)}</span><h2>${esc(c.name)}</h2></div>
    <p class="cn-t">${c.about}</p>
    <div class="cn-f"><span class="st ${cls}">${state}</span><span class="spacer"></span><span class="char-edit">${c.on ? `${icon("tune")}Настроить` : `${icon("plus")}Подключить`}</span></div>
  </article>`;
}

function drawConns() {
  const box = $("#conns");
  if (!box) return;
  const defs = connDefs();
  box.innerHTML = CONN_GROUPS.map(([g, title, sub]) => {
    const list = defs.filter((c) => c.group === g);
    const n = list.filter((c) => c.on).length;
    return list.length ? `<section class="cn-group"><div class="cn-gh"><h2>${title}</h2><span class="meta">${n ? `подключено ${n} из ${list.length}` : sub}</span></div>
      <div class="chars cn-grid">${list.map(connTile).join("")}</div></section>` : "";
  }).join("");
  if (CONN.open) connBody(defs.find((c) => c.id === CONN.open));
  // вход кодом: ждём, пока человек введёт код на сайте
  clearTimeout(drawConns.wait);
  if (defs.some((c) => c.wait)) drawConns.wait = setTimeout(async () => {
    if (location.hash !== "#connections") return;
    const was = { cg: S.chatgpt.signed_in, yt: S.youtube?.account?.signed_in, xm: S.xiaomi?.signed_in, sl: S.smartlife?.signed_in };
    await reload(); drawConns();
    if (!was.sl && S.smartlife?.signed_in) toast("Smart Life подключён — ищу устройства");
    if (!was.xm && S.xiaomi?.signed_in) toast("Xiaomi Home подключён — ищу устройства");
    if (!was.cg && S.chatgpt.signed_in) toast("ChatGPT подключён");
    if (!was.yt && S.youtube?.account?.signed_in) toast("YouTube подключён");
  }, 3000);
}

function connBody(c) {
  const card = $("#cn-card");
  if (!card || !c) return;
  // поля, которые человек уже начал заполнять, перерисовка не стирает
  const keep = {};
  card.querySelectorAll("input, textarea").forEach((el) => { if (el.id && el.value) keep[el.id] = el.value; });
  const net = CONN.net[c.id];
  card.innerHTML = `<div class="cn-mh"><span class="char-av cn-av">${c.icon ? icon(c.icon) : esc(c.mono)}</span>
      <div class="char-n"><h2 id="cn-mt">${esc(c.name)}</h2><span class="st ${c.bad ? "bad" : c.on ? "ok" : c.need ? "warn" : ""}">${c.state}${net?.ok ? ` · ${net.ms} мс` : ""}</span></div>
      <button class="ib" type="button" data-close aria-label="Закрыть">${icon("x")}</button></div>
    <p class="cn-about">${c.about}${c.more ? `. ${c.more}` : ""}</p>
    <div class="cn-body">${c.body}</div>
    ${c.drop ? `<div class="row cn-foot"><span class="spacer"></span>${c.drop}</div>` : ""}`;
  for (const [id, val] of Object.entries(keep)) { const el = card.querySelector("#" + id); if (el) el.value = val; }
}

function connOpen(id) {
  const c = connDefs().find((x) => x.id === id);
  if (!c) return;
  CONN.open = id;
  connBody(c);
  const m = $("#cn-modal");
  m.classList.remove("hidden", "out"); void m.offsetWidth; m.classList.add("in");
  // на телефоне не фокусируем поле: иначе сразу выпрыгивает клавиатура и закрывает окно
  if (matchMedia("(hover: hover)").matches) setTimeout(() => $("#cn-card input, #cn-card textarea, #cn-card .btn.primary")?.focus(), 60);
}

function connClose() {
  const m = $("#cn-modal");
  CONN.open = "";
  m.classList.remove("in"); m.classList.add("out");
  setTimeout(() => { if (m.classList.contains("out")) m.classList.add("hidden"); }, 260);
}

async function connNetTest(btn) {
  await act(btn, async () => {
    if ($("#net-res")) $("#net-res").innerHTML = `<div class="nr"><span class="spin"></span>Проверяю…</div>`;
    let r;
    try { r = await api("POST", "/api/test/net"); } catch (err) { if ($("#net-res")) $("#net-res").innerHTML = ""; throw err; }
    CONN.net = {};
    for (const [name, x] of Object.entries(r)) CONN.net[name] = x;
    drawConns();
    if ($("#net-res")) $("#net-res").innerHTML = Object.entries(r).map(([name, x]) => `<div class="nr ${x.ok ? "ok" : "bad"}"><b>${NET_NAMES[name] || esc(name)}</b>${x.ok ? `работает · ${x.ms} мс` : esc(x.error || "нет связи")}</div>`).join("");
    const bad = Object.entries(r).filter(([, x]) => !x.ok).map(([n]) => NET_NAMES[n] || n);
    if (!Object.keys(r).length) toast("Нечего проверять — нет ни одного ключа", true);
    else if (bad.length) toast(`Нет связи: ${bad.join(", ")}`, true);
    else toast("Связь со всеми есть");
  });
}

async function connClick(e) {
  if (page !== "connections") return;
  const t = e.target;
  const redraw = async () => { await reload(); drawConns(); };
    if(t.closest("[data-ha-remove]")){t.closest("[data-ha-room]").remove();return;}
  if (t.closest("[data-close]")) return connClose();
  if (t.matches("input[data-vpn]")) {
    const svc = t.dataset.vpn, on = t.checked;
    try { await api("PUT", "/api/settings", { vpn: { [svc]: on } }); delete CONN.net[svc]; await redraw(); toast(`${VPN_NAMES.find(([id]) => id === svc)?.[1] || svc}: ${on ? "через VPN" : "напрямую"}`); }
    catch (err) { t.checked = !on; toast(err.message, true); }
    return;
  }
  const gk = t.closest("[data-gk-remove]");
  if (gk) { if (confirm("Убрать этот ключ Google с колонки?")) act(gk, async () => { await api("PUT", "/api/gemini-keys", { remove: +gk.dataset.gkRemove }); await redraw(); }, "Ключ убран"); return; }
  const ek = t.closest("[data-ek-remove], [data-ek-reset]");
  if (ek) {
    const rm = ek.dataset.ekRemove !== undefined, i = +(rm ? ek.dataset.ekRemove : ek.dataset.ekReset);
    if (rm && !confirm(`Убрать ключ ${i + 1} с колонки?`)) return;
    return act(ek, async () => { await api("PUT", "/api/eleven-keys", rm ? { remove: i } : { reset: i }); await redraw(); }, rm ? "Ключ убран" : "Ключ снова в работе");
  }
  const md = t.closest("[data-mem-del]");
  if (md) {
    return act(md, async () => {
      await api("DELETE", "/api/memory", { id: md.dataset.memDel });
      await redraw();
    }, "Удалено из памяти");
  }
  const tile = t.closest(".cn[data-conn]");
  if (tile) return connOpen(tile.dataset.conn);
  const net = t.closest("[data-net]");
  if (net) return connNetTest(net);
  const b = t.closest("[data-key]"), c = t.closest("[data-clear]");
  if (b) {
    const val = $("#k-" + b.dataset.key).value.trim();
    if (!val) return toast("Сначала вставьте ключ в поле", true);
    return act(b, async () => { await api("PUT", "/api/keys", { [b.dataset.key]: val }); $("#k-" + b.dataset.key).value = ""; delete CONN.net[b.dataset.key]; await redraw(); }, (b.dataset.key === "elevenlabs" && (S.eleven_keys || []).length > 1) || (b.dataset.key === "gemini" && (S.gemini_keys || []).length > 1) ? "Ключ добавлен в запас" : "Ключ сохранён");
  }
  if (c) { if (confirm(c.dataset.clear === "elevenlabs" && (S.eleven_keys || []).length > 1 ? "Убрать все ключи ElevenLabs с колонки?" : c.dataset.clear === "gemini" && (S.gemini_keys || []).length > 1 ? "Убрать все ключи Google с колонки?" : "Убрать этот ключ с колонки?")) act(c, async () => { await api("PUT", "/api/keys", { [c.dataset.clear]: "" }); delete CONN.net[c.dataset.clear]; await redraw(); }); return; }
  const btn = t.closest("button[id]");
  if (!btn) return;
  switch (btn.id) {
    case "ha-discover-rooms": return act(btn,async()=>{
      const r=await api("POST","/api/home-assistant/rooms/discover",{});
      if(!r.rooms?.length){toast("Автоматическое определение комнат недоступно. Можно выбрать готовые сценарии уборки.",true);return;}
      const catalog=await api("GET","/api/home-assistant");CONN.haEntities=catalog.entities || [];
      $("#ha-rooms").innerHTML=r.rooms.map(haRoomRow).join("");toast(`Найдено комнат: ${r.rooms.length}. Нажмите «Проверить и сохранить».`);
    });
    case "ha-add-room": $("#ha-rooms").insertAdjacentHTML("beforeend",haRoomRow()); return;
    case "ha-load": return act(btn,async()=>{
      const r=await api("GET","/api/home-assistant");CONN.haEntities=r.entities || [];
      const rooms=$$("[data-ha-room]").map(el=>({...JSON.parse(el.dataset.haBinding || "{}"),name:el.querySelector(".ha-name").value,vacuum:el.querySelector(".ha-vacuum").value,entity:el.querySelector(".ha-entity").value,room_ids:el.querySelector(".ha-ids").value}));
      $("#ha-rooms").innerHTML=rooms.map(haRoomRow).join("");
      toast(`Получено устройств и действий: ${CONN.haEntities.length}`);
    });
    case "ha-save": return act(btn,async()=>{
      const rooms=$$("[data-ha-room]").map(el=>({...JSON.parse(el.dataset.haBinding || "{}"),name:el.querySelector(".ha-name").value.trim(),vacuum:el.querySelector(".ha-vacuum").value,entity:el.querySelector(".ha-entity").value,room_ids:el.querySelector(".ha-ids").value.trim()}));
      await api("PUT","/api/home-assistant",{url:$("#ha-url").value,token:$("#ha-token").value,rooms});
      $("#ha-token").value="";await redraw();
    },"Home Assistant подключён");
    case "ha-disconnect": return act(btn,async()=>{await api("DELETE","/api/home-assistant");CONN.haEntities=[];await redraw();},"Home Assistant отключён");
    case "cg-in": return act(btn, async () => { await api("POST", "/api/chatgpt/login"); await redraw(); });
    case "cg-newchat": if (confirm("Начать новый чат? ChatGPT забудет прошлые разговоры.")) act(btn, async () => { await api("POST", "/api/chatgpt/newchat"); await redraw(); }, "Новый чат начат"); return;
    case "cg-out": if (confirm("Выйти из ChatGPT на колонке?")) act(btn, async () => { await api("DELETE", "/api/chatgpt"); await redraw(); }); return;
    case "sl-in": {
      const code = $("#sl-code").value.trim();
      if (!code) return toast("Сначала вставьте код пользователя из Smart Life", true);
      return act(btn, async () => { await api("POST", "/api/smartlife/login", { user_code: code }); await redraw(); });
    }
    case "sl-refresh": return act(btn, async () => { const r = await api("POST", "/api/smartlife/refresh"); await redraw(); toast(`Устройств: ${r.devices}`); });
    case "sl-out": if (confirm("Выйти из Smart Life на колонке?")) act(btn, async () => { await api("DELETE", "/api/smartlife"); await redraw(); }); return;
    case "xm-in": return act(btn, async () => { await api("POST", "/api/xiaomi/login"); await redraw(); });
    case "xm-refresh": return act(btn, async () => { const r = await api("POST", "/api/xiaomi/refresh"); await redraw(); toast(`Устройств: ${r.devices}`); });
    case "xm-out": if (confirm("Выйти из Xiaomi Home на колонке?")) act(btn, async () => { await api("DELETE", "/api/xiaomi"); await redraw(); }); return;
    case "ya-in": return act(btn, async () => { await api("POST", "/api/youtube/login"); await redraw(); });
    case "ya-out": if (confirm("Выйти из YouTube на колонке?")) act(btn, async () => { await api("DELETE", "/api/youtube/login"); await redraw(); }); return;
    case "yt-save": {
      const txt = $("#yt-cookies").value.trim();
      if (!txt) return toast("Сначала вставьте текст файла cookies.txt", true);
      return act(btn, async () => { await api("PUT", "/api/youtube", { cookies: txt }); $("#yt-cookies").value = ""; await redraw(); }, "YouTube принял вход");
    }
    case "yt-clear": if (confirm("Убрать вход в YouTube с колонки?")) act(btn, async () => { await api("DELETE", "/api/youtube"); await redraw(); }); return;
    case "vless-save": return act(btn, async () => { await api("PUT", "/api/vless", { link: $("#vless").value }); $("#vless").value = ""; await redraw(); }, "Ссылка сохранена — проверьте связь");
    case "vless-clear": if (confirm("Убрать VPN? Колонка пойдёт в интернет напрямую.")) act(btn, async () => { await api("PUT", "/api/vless", { link: "" }); await redraw(); }); return;
    case "tg-save": return act(btn, async () => {
      const token = $("#tg-token")?.value.trim();
      const chatId = $("#tg-chat-id")?.value.trim();
      const enabled = $("#tg-enabled")?.checked ?? true;
      const body = { chat_id: chatId, enabled: enabled };
      if (token) body.token = token;
      await api("PUT", "/api/telegram", body);
      if ($("#tg-token")) $("#tg-token").value = "";
      await redraw();
    }, "Настройки Telegram сохранены");
    case "tg-test": return act(btn, async () => {
      const token = $("#tg-token")?.value.trim();
      const chatId = $("#tg-chat-id")?.value.trim();
      await api("POST", "/api/telegram/test", { token: token, chat_id: chatId });
    }, "Тестовое сообщение отправлено в Telegram!");
    case "tg-clear": if (confirm("Убрать настройки Telegram-бота с колонки?")) act(btn, async () => {
      await api("DELETE", "/api/telegram");
      await redraw();
    }, "Telegram отключён"); return;
    case "mem-add": {
      const val = $("#mem-new")?.value.trim();
      if (!val) return toast("Введите факт для сохранения", true);
      return act(btn, async () => {
        await api("POST", "/api/memory", { fact: val });
        if ($("#mem-new")) $("#mem-new").value = "";
        await redraw();
      }, "Факт сохранён в память");
    }
  }
}

// ---------------------------------------------------------------- колонка
const TZ = [["Europe/Kaliningrad", "Калининград (UTC+2)"], ["Europe/Moscow", "Москва (UTC+3)"], ["Europe/Minsk", "Минск (UTC+3)"],
  ["Europe/Kyiv", "Киев"], ["Europe/Samara", "Самара (UTC+4)"], ["Asia/Yekaterinburg", "Екатеринбург (UTC+5)"],
  ["Asia/Almaty", "Алматы (UTC+5)"], ["Asia/Omsk", "Омск (UTC+6)"], ["Asia/Novosibirsk", "Новосибирск (UTC+7)"],
  ["Asia/Krasnoyarsk", "Красноярск (UTC+7)"], ["Asia/Irkutsk", "Иркутск (UTC+8)"], ["Asia/Yakutsk", "Якутск (UTC+9)"],
  ["Asia/Vladivostok", "Владивосток (UTC+10)"], ["Asia/Magadan", "Магадан (UTC+11)"], ["Asia/Kamchatka", "Камчатка (UTC+12)"],
  ["Europe/Berlin", "Берлин"], ["Europe/London", "Лондон"], ["UTC", "UTC"]];
const HOME_USE = [
  ["alice", "Алиса", "Джарвис передаёт команду штатной Алисе — она умеет всё, что подключено к Яндексу", () => true],
  ["yandexmusic", "Яндекс Музыка", "«Включи музыку», «поставь песню…» Джарвис отдаёт Алисе — играет из Яндекс Музыки. Выключено — всё играет с YouTube", () => true],
  ["smartlife", "Smart Life", "Напрямую по домашней сети — быстрее всего", () => S.smartlife?.signed_in],
  ["xiaomi", "Xiaomi Home", "Через облако Xiaomi: пылесос, увлажнитель, лампы Mi Home", () => S.xiaomi?.signed_in],
  ["yandex", "Яндекс Умный дом", "Сам Джарвис через API Яндекса, без Алисы — нужен токен", () => !!S.keys?.yandex],
  ["homeassistant", "Home Assistant", "Ваш сервер Home Assistant", () => S.home_assistant?.configured],
];
function homeUseRows() {
  const use = S.settings.home_use || {};
  return HOME_USE.map(([id, name, about, ready]) => { const on = ready(); return `<label class="sw hu${on ? "" : " off"}"><input type="checkbox" data-home="${id}"${on && use[id] !== false ? " checked" : ""}${on ? "" : " disabled"}><span class="tr"></span>
    <span class="hu-t"><b>${name}</b><small>${about}${on ? "" : ` · <a href="#connections">сначала подключите</a> — тогда галочка включится сама`}</small></span></label>`; }).join("");
}
function viewSpeaker() {
  const s = S.settings, via = s.home_via === "jarvis" ? "jarvis" : "alice";
  $("#view").innerHTML = `<header class="ph"><h1>Колонка</h1><p class="sub">Изменения сохраняются сразу</p></header>
  ${sec("Умный дом и музыка", "Через что Джарвис может управлять домом и откуда включать музыку. Можно оставить несколько: быстрые команды сначала пробуют Smart Life по домашней сети, потом Алису, а сложные просьбы модель решает сама теми способами, что здесь включены",
    `<div class="home-use">${homeUseRows()}</div>`)}
  ${sec("Где вы", "Комната колонки — для команд без комнаты («выключи свет»). Город и пояс — для погоды, новостей и времени: часы колонки идут по Гринвичу",
    `<div class="f"><label for="s-room">Комната, где стоит колонка ${tip("«Выключи свет», «включи кондиционер» без названия комнаты — Джарвис сделает это здесь, как Алиса.")}</label>
      <input type="text" id="s-room" list="s-rooms" value="${esc(s.speaker_room || "")}" placeholder="Например, Зал" autocomplete="off">
      <datalist id="s-rooms">${(S.rooms || []).map((r) => `<option value="${esc(r)}">`).join("")}</datalist></div>
    <div class="grid g2"><div class="f"><label for="s-city">Город</label><input type="text" id="s-city" value="${esc(s.city || "")}" placeholder="Например, Москва"></div>
      <div class="f"><label for="s-tz">Часовой пояс</label><select id="s-tz">${TZ.map(([id, n]) => `<option value="${id}"${id === (s.timezone || "UTC") ? " selected" : ""}>${n}</option>`).join("")}</select></div></div>`)}
  ${sec("Слово «Джарвис»", "Его распознаёт сама колонка, без интернета",
    `<label class="sw"><input type="checkbox" id="s-wake"${s.wake_enabled ? " checked" : ""}><span class="tr"></span>Откликаться на «Джарвис» ${tip("Выключите, если Джарвис нужен только по кнопке «Поговорить». Алиса работает в любом случае.")}</label>
    <label class="sw"><input type="checkbox" id="s-check"${s.wake_check ? " checked" : ""}><span class="tr"></span>Проверять имя перед ответом ${tip("Выключено — как у Алисы: услышал слово, сразу загорается кольцо и Джарвис слушает. Включено — сначала облако проверяет, что прозвучало именно «Джарвис»: ложных разговоров меньше, но отклик на полсекунды-секунду позже.")}</label>
    <div class="f"><label>Чувствительность ${tip("Выше — слышит издалека и тихо, но чаще срабатывает зря (например, на разговор в Discord). Ниже — реже ошибается, но звать нужно чётче.")}<b id="s-sens-v" class="val">${s.wake_sensitivity}</b></label>
      <input type="range" id="s-sens" min="1" max="10" value="${s.wake_sensitivity}"><div class="scale"><span>осторожно</span><span>слышит всё</span></div></div>
    <div class="f"><label>Общее усиление микрофона ${tip("Для колонок, которые слышат вас только вблизи (так бывает на Станции Мини 2: её обработка отдаёт голос тихо). Усиливает звук для всего: слова «Джарвис», ИИ и записей для обучения. Прибавляйте по 6 дБ и проверяйте; слишком много — чаще ложные срабатывания. На Лайт обычно не нужно.")}<b id="s-gain-v" class="val">+${s.mic_gain_db || 0} дБ</b></label>
      <input type="range" id="s-gain" min="0" max="30" step="3" value="${s.mic_gain_db || 0}"><div class="scale"><span>как есть</span><span>+30 дБ</span></div></div>
    <div class="f"><label>Усиление микрофона для ИИ ${tip("Если модель плохо слышит и переспрашивает, хотя вы говорите не тихо (так бывает на Станции Лайт 2: её обработка отдаёт голос тише), прибавьте 6–12 дБ. Слово «Джарвис» это не меняет. Уровень голоса за каждый разговор пишется в журнал: «микрофон: за разговор голос …».")}<b id="s-mic-v" class="val">+${s.mic_boost_db || 0} дБ</b></label>
      <input type="range" id="s-mic" min="0" max="18" step="3" value="${s.mic_boost_db || 0}"><div class="scale"><span>как есть</span><span>+18 дБ</span></div></div>
    ${S.speaker.wake_model ? "" : `<p class="note">Модель слова ещё не установлена — пока Джарвиса можно позвать только кнопкой «Поговорить» на главной.</p>`}`)}
  ${sec("Язык", "На каком языке вы говорите с Джарвисом. Распознавание речи и ответы будут на нём — так он не «услышит» испанский там, где был русский",
    `<div class="seg" id="s-lang">${[["ru", "Русский"], ["en", "English"], ["uk", "Українська"], ["", "Определять самому"]].map(([k, n]) => `<button type="button" data-lang="${k}" class="${(s.language ?? "ru") === k ? "on" : ""}">${n}</button>`).join("")}</div>`)}
  ${sec(`<span class="moon">${icon("moon")}</span>Ночной режим`, "По расписанию Джарвис говорит тише и короче, кольцо светит слабее, а сама колонка — Алиса, музыка, YouTube — не играет громче заданного",
    `<label class="sw"><input type="checkbox" id="n-on"${s.night?.enabled ? " checked" : ""}><span class="tr"></span>Включать по расписанию</label>
    <div class="night-t"><div class="f"><label>С</label><input type="time" id="n-from" value="${esc(s.night?.from || "23:00")}"></div><div class="f"><label>До</label><input type="time" id="n-to" value="${esc(s.night?.to || "08:00")}"></div>${s.night_now ? `<span class="badge acc">сейчас ночь</span>` : ""}</div>
    <div class="night-sl"><div class="f"><label>Голос Джарвиса не громче <b class="val" id="n-voice-v"></b></label><div id="n-voice"></div></div>
      ${S.speaker.leds ? `<div class="f"><label>Кольцо не ярче <b class="val" id="n-bright-v"></b></label><div id="n-bright"></div></div>` : ""}
      <div class="f"><label>Колонка не громче ${tip("Громкость самой колонки по её шкале 0–10. Если ночью сделать громче, через полминуты станет обратно.")}<b class="val" id="n-speaker-v"></b></label><div id="n-speaker"></div></div></div>`)}
  ${sec("Подсветка", "Кольцо Джарвиса. Цвета у каждого ассистента свои — в его настройках. У Алисы подсветка своя и не меняется",
    S.speaker.leds ? `<label class="sw"><input type="checkbox" id="l-on"${s.led.enabled ? " checked" : ""}><span class="tr"></span>Подсвечивать кольцо, когда говорит Джарвис</label>
    ${ledLabHtml()}`
    : `<p class="note">У этой колонки нет подсветки, которой может управлять Джарвис.</p>`)}`;
  $("#view").oninput = $("#view").onchange = $("#view").onclick = null;
  const put = (body) => api("PUT", "/api/settings", body).then(reload).catch((e) => toast(e.message, true));
  $$("input[data-home]").forEach((b) => b.addEventListener("change", () => {
    const ch = b.dataset.home, on = b.checked, name = HOME_USE.find((x) => x[0] === ch)?.[1] || ch;
    put({ home_use: { [ch]: on } }).then(() => toast(`${name}: ${on ? "используется" : "не используется"}`));
  }));
  $("#s-wake").addEventListener("change", (e) => put({ wake_enabled: e.target.checked }).then(() => toast(e.target.checked ? "Джарвис снова слушает" : "Джарвис больше не откликается на слово")));
  $("#s-check").addEventListener("change", (e) => put({ wake_check: e.target.checked }).then(() => toast(e.target.checked ? "Сначала проверяю имя, потом отвечаю" : "Отклик сразу, как у Алисы")));
  $("#s-city").addEventListener("change", (e) => put({ city: e.target.value.trim() }).then(() => toast("Город сохранён")));
  $("#s-room").addEventListener("change", (e) => put({ speaker_room: e.target.value.trim() }).then(() => toast(e.target.value.trim() ? `Колонка стоит: ${e.target.value.trim()}` : "Комната колонки не задана")));
  $("#s-tz").addEventListener("change", (e) => put({ timezone: e.target.value }).then(() => toast("Часовой пояс сохранён — время в журнале тоже по нему")));
  $("#s-sens").addEventListener("input", (e) => ($("#s-sens-v").textContent = e.target.value));
  $("#s-sens").addEventListener("change", (e) => put({ wake_sensitivity: +e.target.value }).then(() => toast("Чувствительность сохранена")));
  $("#s-gain").addEventListener("input", (e) => { $("#s-gain-v").textContent = `+${e.target.value} дБ`; });
  $("#s-gain").addEventListener("change", (e) => put({ mic_gain_db: +e.target.value }).then(() => toast("Общее усиление микрофона сохранено")));
  $("#s-mic").addEventListener("input", (e) => { $("#s-mic-v").textContent = `+${e.target.value} дБ`; });
  $("#s-mic").addEventListener("change", (e) => put({ mic_boost_db: +e.target.value }).then(() => toast("Усиление микрофона сохранено")));
  const night = { enabled: false, from: "23:00", to: "08:00", voice: 20, bright: 10, speaker: 3, ...(s.night || {}) };
  const saveNight = () => put({ night }).then(() => toast(night.enabled ? `Ночной режим: ${night.from}–${night.to}` : "Ночной режим выключен"));
  $("#n-on").addEventListener("change", (e) => { night.enabled = e.target.checked; saveNight(); });
  $("#n-from").addEventListener("change", (e) => { if (e.target.value) { night.from = e.target.value; saveNight(); } });
  $("#n-to").addEventListener("change", (e) => { if (e.target.value) { night.to = e.target.value; saveNight(); } });
  const nsl = (key, label, min, max, step, fmt) => {
    const out = $(`#n-${key}-v`), show = (v) => { out.textContent = fmt(v); };
    if (!$(`#n-${key}`)) return;
    show(night[key]);
    jelly($(`#n-${key}`), { label, min, max, step, value: night[key], format: fmt, onInput: show, onChange: (v) => { show(v); night[key] = v; saveNight(); } });
  };
  nsl("voice", "Голос ночью", 5, 100, 5, (v) => v + "%");
  nsl("bright", "Кольцо ночью", 0, 100, 5, (v) => v + "%");
  nsl("speaker", "Колонка ночью", 1, 10, 1, (v) => `${v} из 10`);
  $$("#s-lang [data-lang]").forEach((b) => b.addEventListener("click", () => {
    $$("#s-lang button").forEach((x) => x.classList.toggle("on", x === b));
    put({ language: b.dataset.lang }).then(() => toast(b.dataset.lang ? `Язык: ${b.textContent}` : "Язык определяется сам"));
  }));
  if (S.speaker.leds) {
    ledLabBind();
    $("#l-on").addEventListener("change", (e) => put({ led: { enabled: e.target.checked } }).then(() => toast(e.target.checked ? "Подсветка включена" : "Подсветка выключена")));
  }
}

// ---------------------------------------------------------------- безопасность
function secHead(tab) {
  return `<header class="ph"><h1>${tab === "training" ? "Обучение" : "Безопасность"}</h1><p class="sub">${tab === "training" ? "Помогите Джарвису лучше узнавать своё имя — по вашему голосу и вашей комнате" : "Кто может попасть в колонку и её настройки"}</p></header>`;
}

function viewSecurity() {
  $("#view").innerHTML = `${secHead("security")}
  ${sec("Пароль настроек", "После смены все остальные браузеры выйдут из настроек",
    `<div class="grid g2"><div class="f"><label>Текущий</label><input type="password" id="pw-old" autocomplete="current-password"></div>
      <div class="f"><label>Новый ${tip("Минимум 6 символов.")}</label><input type="password" id="pw-new" autocomplete="new-password"></div></div>
    <div><button class="btn" id="pw-save">Сменить пароль</button></div>`)}
  ${sec("Доступ к настройкам", "После 5 неверных паролей вход временно блокируется. Ключи и ссылка VLESS не уходят обратно в браузер",
    `<label class="sw"><input type="checkbox" id="lan"${S.settings.lan_only ? " checked" : ""}><span class="tr"></span>Только из домашней сети ${tip("Настройки открываются только с устройств в вашей Wi-Fi сети. Выключайте, только если точно понимаете зачем (например, доступ через свой VPN).")}</label>`)}
  ${sec("Что открыто на колонке", "Служебные входы, через которые в колонку можно попасть из домашней сети",
    `<div class="checks">${S.security.map((c) => `<div class="check"><div class="body"><b>${esc(c.title)}</b><small>${esc(c.hint)}</small></div><span class="st ${c.open ? "warn" : "ok"}">${c.open ? "открыт" : "закрыт"}</span></div>`).join("") || `<p class="note">Проверять нечего</p>`}</div>`)}
  ${sec("Обновления", "", `<div id="upd"><p class="note">…</p></div>`)}
  ${sec("Для разработчика", "", `<label class="sw"><input type="checkbox" id="dev"${S.settings.dev_mode ? " checked" : ""}><span class="tr"></span><span>Режим разработчика <small class="muted">— вкладка «Обучение»: дообучение слова «Джарвис» и запись фона</small></span></label>`)}`;
  drawUpdate();
  $("#dev").addEventListener("change", (e) => act(null, async () => { await api("PUT", "/api/settings", { dev_mode: e.target.checked }); await reload(); viewSecurity(); }, e.target.checked ? "Режим разработчика включён — вкладка «Обучение» рядом" : "Режим разработчика выключен"));
  $("#view").oninput = $("#view").onchange = $("#view").onclick = null;
  $("#pw-save").addEventListener("click", (e) => act(e.target, async () => { await api("PUT", "/api/password", { old: $("#pw-old").value, new: $("#pw-new").value }); $("#pw-old").value = $("#pw-new").value = ""; }, "Пароль сменён"));
  $("#lan").addEventListener("change", async (e) => {
    if (!e.target.checked && !confirm("Разрешить вход в настройки не только из домашней сети?")) { e.target.checked = true; return; }
    await act(null, async () => { await api("PUT", "/api/settings", { lan_only: e.target.checked }); await reload(); });
  });
}

// ---------------------------------------------------------------- загрузка файлов (тело — сам файл)
async function upload(method, url, file) {
  const r = await fetch(url, { method, headers: { "X-Jarvis": "1", "Content-Type": "application/octet-stream" }, body: file, credentials: "same-origin" });
  let d = {};
  try { d = await r.json(); } catch { /* пусто */ }
  if (r.status === 401 || r.status === 428) { showGate(d.setup); throw new Error(d.error || "Нужно войти"); }
  if (!r.ok) throw new Error(d.error || `Ошибка ${r.status}`);
  return d;
}

// после обновления или новой модели колонка перезапускается — ждём, пока поднимется
async function waitRestart(msg) {
  toast(msg);
  const old = S.version;
  for (let i = 0; i < 60; i++) {
    await new Promise((r) => setTimeout(r, 3000));
    try { await reload(); if (S.version !== old) { location.reload(); return; } toast("Колонка перезапустилась"); route(); return; } catch { /* ещё перезапускается */ }
  }
  toast("Колонка долго не отвечает — обновите страницу через минуту", true);
}

// коды входа (YouTube, Google Диск, ChatGPT…) копируются нажатием
document.addEventListener("click", (e) => {
  const c = e.target.closest?.(".code-big");
  if (c) copyText(c.textContent.trim()).then(() => toast("Код скопирован"), () => toast("Не получилось скопировать", true));
});
// скопировать текст: на http (адрес колонки в сети) navigator.clipboard недоступен — запасной способ
async function copyText(text) {
  try { await navigator.clipboard.writeText(text); return; } catch { /* ниже */ }
  const t = document.createElement("textarea");
  t.value = text; t.setAttribute("readonly", ""); t.style.position = "fixed"; t.style.opacity = "0";
  document.body.appendChild(t); t.select();
  try { document.execCommand("copy"); } finally { t.remove(); }
}

// ---------------------------------------------------------------- плеер на главной (Glass player, Kokonut UI «Now Playing», MIT)
// Пластинка с обложкой видео: разгоняется и останавливается по инерции (τ 0,22 / 0,2 с), 33⅓ об/мин — 200°/с;
// её можно крутить пальцем — это перемотка. Тонарм опускается за 500 мс, эквалайзер дышит в такт,
// цифры времени перекатываются (220 мс). Карточка всегда одной высоты.
let M = null, mAt = 0;
const NP = { angle: 0, omega: 0, env: 0, held: false, scrub: null, lastA: 0, lastT: 0, samples: [], seekHeld: null };
const CRUISE = 200, SPIN_UP = 0.22, SPIN_DOWN = 0.2, COAST = 0.45;
const clockStr = (s) => { s = Math.max(0, Math.floor(s || 0)); return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`; };
const mPlaying = () => !!M?.playing && !M.paused;
const mPos = () => !M ? 0 : NP.scrub != null ? NP.scrub : NP.seekHeld != null ? NP.seekHeld : mPlaying() ? Math.min(M.dur || Infinity, M.pos + (Date.now() - mAt) / 1000) : M.pos;
const hash = (n) => { const s = Math.sin(n * 127.1) * 43758.5453; return s - Math.floor(s); };
const noise = (x) => { const i = Math.floor(x), f = x - i, u = f * f * (3 - 2 * f); return hash(i) * (1 - u) + hash(i + 1) * u; };

const ARM = `<svg class="np-arm" viewBox="0 0 40 110" aria-hidden="true"><circle class="a1" cx="31" cy="11" r="8"/><circle class="a2" cx="31" cy="11" r="3"/><path class="a3" d="M31 11 L31 70 L18 92"/><rect class="a4" height="12" rx="1.5" transform="rotate(32 16 96)" width="7" x="12.5" y="90"/></svg>`;

function playerHtml() {
  return `<section class="card np" id="np" data-state="idle">
    <div class="mu-top">
      <a class="cover" href="#player" id="np-cover" aria-label="Открыть плеер">${icon("music")}<img id="np-art" alt=""></a>
      <div class="np-info"><div class="np-src" id="np-kicker"></div><div class="np-title" id="np-title">Тишина</div><div class="np-artist" id="np-artist">Включите что-нибудь с YouTube или попросите Алису</div></div>
    </div>
    <div class="progress np-play-ui"><span class="roll" id="np-cur"></span><div id="np-seek"></div><span id="np-dur"></span></div>
    <div class="controls np-play-ui">
      <button class="cbtn" type="button" data-m="stop" aria-label="Выключить">${icon("stop", "solid")}</button>
      <button class="cbtn" type="button" data-m="prev" id="np-prev" aria-label="Предыдущий">${icon("prev", "solid")}</button>
      <button class="cbtn big" type="button" data-m="toggle" id="np-toggle" aria-label="Пауза">${icon("play", "solid")}</button>
      <button class="cbtn" type="button" data-m="next" id="np-next" aria-label="Следующий">${icon("skip", "solid")}</button>
      <button class="cbtn" type="button" data-m="copy" aria-label="Скопировать ссылку">${icon("link")}</button>
    </div>
    <form class="np-form" id="np-form"><input type="text" id="np-q" placeholder="Название или ссылка на YouTube" aria-label="Что включить" autocomplete="off"><button class="cbtn big" type="submit" aria-label="Включить">${icon("play", "solid")}</button></form>
  </section>
  ${volCardHtml()}`;
}

// громкость колонки — отдельная карточка (на главной и на странице «Плеер»)
function volCardHtml() {
  return `<section class="card vol-card" id="np-vol">
    <div class="card-head"><h2>Громкость колонки</h2><span class="meta">Алиса, музыка и Джарвис</span></div>
    <div class="vol-main">
      <div class="vol-num"><b class="vol-big" id="vol-big">–</b><small>из 10</small></div>
      <div class="vol-ctl">
        <button class="cbtn" type="button" data-vol="-1" aria-label="Тише">${icon("minus")}</button>
        <div class="vol-sl"><div id="vol"></div><div class="vol-ticks" aria-hidden="true">${Array.from({ length: 11 }, (_, i) => `<i data-n="${i}"></i>`).join("")}</div></div>
        <button class="cbtn" type="button" data-vol="1" aria-label="Громче">${icon("plus")}</button>
      </div>
    </div>
  </section>`;
}

async function musicData() {
  try { M = await api("GET", "/api/music"); mAt = Date.now(); } catch { return; }
  drawPlayer(); drawPlayerPage();
  $("#nav-eq")?.classList.toggle("hidden", !mPlaying());
}

// обложка: YouTube — крупная (если есть), иначе средняя; Яндекс Музыка — как пришла
function setArt(img, art, t) {
  if (img.dataset.src === art) return;
  img.dataset.src = art; img.classList.remove("ok");
  if (!art) { img.removeAttribute("src"); return; }
  img.onload = () => { if (img.naturalWidth <= 120 && img.src.includes("maxres")) { img.src = artOf(t, false); return; } img.classList.add("ok"); };
  img.onerror = () => { if (img.src.includes("maxres")) img.src = artOf(t, false); };
  img.src = art;
}

function drawPlayer() {
  const np = $("#np");
  if (!np || !M) return;
  const t = M.track || {}, on = M.playing || M.paused, ya = M.source === "yandex";
  np.dataset.state = M.loading && !on ? "loading" : !on ? "idle" : M.paused ? "paused" : "playing";
  setArt($("#np-art"), on ? artOf(t, true) : "", t);
  $("#np-kicker").innerHTML = on ? `<span class="np-eq" aria-hidden="true"><i></i><i></i><i></i></span>${ya ? "Яндекс Музыка" : "YouTube"}${M.paused ? " · пауза" : ""}` : M.loading ? "ищу…" : "Плеер";
  swapText($("#np-title"), on ? t.Title || "…" : M.loading ? "Ищу и включаю…" : "Тишина");
  swapText($("#np-artist"), on ? t.Author || (ya ? "Яндекс Музыка" : "YouTube") : "Включите что-нибудь с YouTube или попросите Алису");
  $("#np-toggle").innerHTML = icon(M.paused ? "play" : "pause", "solid");
  $("#np-toggle").setAttribute("aria-label", M.paused ? "Продолжить" : "Пауза");
  $("#np-prev").disabled = !ya && (M.history || []).length < 2;
  $("#np-next").disabled = !ya && !(M.next || []).length;
  $("#np-dur").textContent = M.dur ? clockStr(M.dur) : "";
  Motion.add(npAnim);
}

// управление музыкой — общее для главной и страницы «Плеер»
function musicAct(a, b) {
  if (!M) return;
  if (b) { b.classList.remove("k-prev", "k-next", "k-pop"); void b.offsetWidth; b.classList.add(a === "prev" ? "k-prev" : a === "next" ? "k-next" : "k-pop"); }
  if (a === "copy") { const l = musicLink(); if (l) copyText(l).then(() => toast("Ссылка скопирована"), () => toast("Не получилось скопировать", true)); return; }
  if (a === "radio") { music({ action: "radio", on: !M.radio }, b).then((ok) => ok && toast(M.radio ? "Радио: после очереди — похожее" : "Радио выключено: после очереди тишина")); return; }
  if (a === "prev" && M.source !== "yandex") { const h = (M.history || [])[1]; if (h) music({ action: "id", query: h.ID }, b); return; }
  music({ action: a === "toggle" ? (M.paused ? "resume" : "pause") : a }, b);
}

// перекатывающиеся цифры: сменившаяся уезжает вверх, новая въезжает снизу (назад — наоборот)
function rollTo(el, text, dir) {
  if (el.dataset.v === text) return;
  const old = el.dataset.v || "";
  el.dataset.v = text;
  if (Motion.reduce || old.length !== text.length) { el.innerHTML = [...text].map((ch) => `<span class="rd"><span>${ch}</span></span>`).join(""); return; }
  [...text].forEach((ch, i) => {
    const slot = el.children[i];
    if (!slot || old[i] === ch) return;
    slot.innerHTML = `<span class="out ${dir > 0 ? "up" : "down"}">${old[i]}</span><span class="in ${dir > 0 ? "up" : "down"}">${ch}</span>`;
  });
}

const npAnim = {
  lastSec: -1,
  t: 0,
  frame(dt) { this.t += dt; },
  render() {
    const p = mPos(), t = Motion.reduce ? 0.37 : this.t, env = mPlaying() ? 1 : 0;
    $$(".np-eq i").forEach((bar, i) => {
      const lvl = 0.18 + 0.82 * (0.6 * noise(t * 4.3 + i * 17.3) + 0.4 * noise(t * 9.1 + i * 31.7 + 100)) * env;
      bar.style.transform = `scaleY(${lvl.toFixed(3)})`;
    });
    if (seekJ && M && M.dur && !seekJ.dragging) { seekJ.range(0, M.dur); seekJ.set(p); }
    const sec = Math.floor(p), cur = $("#np-cur") || $("#pl-cur");
    if (cur && (sec !== this.lastSec || cur !== this.lastEl)) { rollTo(cur, M && (M.playing || M.paused) ? clockStr(p) : "", sec >= this.lastSec ? 1 : -1); this.lastSec = sec; this.lastEl = cur; }
  },
  done() { return (!$("#np") && !$("#plh")) || !mPlaying(); },
};

function playerSetup() {
  const np = $("#np");
  np.addEventListener("click", (e) => {
    const b = e.target.closest("[data-m]");
    if (!b || b.disabled) return;
    musicAct(b.dataset.m, b);
  });
  $("#np-form").addEventListener("submit", (e) => {
    e.preventDefault();
    const q = $("#np-q").value.trim();
    if (!q) return;
    $("#np-q").value = "";
    M = { ...(M || {}), loading: true }; drawPlayer();
    music({ action: "play", query: q });
  });
  makeSliders($("#np-seek"), $("#vol"));
}

// перемотка и громкость — «желейные» ползунки (на главной и в плеере)
function makeSliders(seekEl, volEl) {
  seekJ = jelly(seekEl, { label: "Перемотка", min: 0, max: Math.max(1, M?.dur || 1), value: 0, format: clockStr, keyStep: 5,
    disabled: () => !M?.dur,
    onInput: (v) => { NP.seekHeld = v; Motion.add(npAnim); },
    onChange: (v) => { NP.seekHeld = null; if (!M?.dur) return; M.pos = v; mAt = Date.now(); Motion.add(npAnim); music({ action: "seek", sec: Math.round(v) }); } });
  volJ = jelly(volEl, { label: "Громкость колонки", min: 0, max: 10, step: 1, value: S.speaker.volume > 0 ? S.speaker.volume : 5, format: (v) => String(v),
    onInput: (v) => { volDrag = true; volShow(v); },
    onChange: (v) => setVolume(v) });
  const card = volEl.closest(".vol-card");
  card?.addEventListener("click", (e) => {
    const b = e.target.closest("[data-vol]");
    if (!b) return;
    const cur = volWant ?? (S.speaker.volume >= 0 ? S.speaker.volume : 5);
    const v = Motion.clamp(cur + Number(b.dataset.vol), 0, 10);
    b.classList.remove("k-pop"); void b.offsetWidth; b.classList.add("k-pop");
    if (v === cur) return;
    volJ.set(v, true); volShow(v); setVolume(v);
  });
  if (S.speaker.volume >= 0) volShow(S.speaker.volume);
}

// число и деления под ползунком; отправка с задержкой — чтобы частые «+» не долбили колонку
let volWant = null, volTimer = 0;
function volShow(v) {
  const big = $("#vol-big"); if (big) { big.textContent = v; big.dataset.v = v; }
  $$(".vol-ticks i").forEach((t) => t.classList.toggle("on", Number(t.dataset.n) <= v));
  const c = $("#np-vol"); if (c) c.dataset.lvl = v === 0 ? "mute" : v <= 3 ? "low" : v <= 7 ? "mid" : "high";
}
function setVolume(v) {
  volDrag = true; volWant = v; volShow(v);
  clearTimeout(volTimer);
  volTimer = setTimeout(() => {
    api("POST", "/api/speaker/volume", { level: v }).then((r) => { S.speaker.volume = r.volume; })
      .catch((e) => toast(e.message, true)).finally(() => { if (volWant === v) { volWant = null; volDrag = false; } });
  }, 250);
}

async function music(body, btn) {
  let ok = false;
  await act(btn, async () => { M = await api("POST", "/api/music", body); mAt = Date.now(); ok = true; });
  drawPlayer(); drawPlayerPage();
  $("#nav-eq")?.classList.toggle("hidden", !mPlaying());
  return ok;
}

// громкость колонки (её слышно у Алисы, музыки и Джарвиса); неизвестна — ползунок прячем
let volDrag = false, seekJ = null, volJ = null;
function drawVolume() {
  const box = $("#np-vol");
  if (!box || !volJ || volDrag) return;
  const v = S.speaker.volume;
  box.classList.toggle("hidden", v == null || v < 0);
  if (v != null && v >= 0) { volJ.set(v, true); volShow(v); }
}

// ---------------------------------------------------------------- обучение слова
const KIND = { pos: "Джарвис", false: "Не Джарвис", near: "Не проверено", bg: "Фон" };
const GUESS = { pos: "колонка решила, что звали", false: "колонка сработала, но решила, что зря", near: "колонка насторожилась, но не сработала" };
let T = null, skipped = new Set(), clipFilter = "all", clipLimit = 40;

const clipUrl = (x) => `/api/train/clip?kind=${x.kind}&name=${encodeURIComponent(x.name)}`;
const clipWhen = (x) => new Date(x.time).toLocaleString("ru-RU", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
const clipProb = (x) => (x.name.match(/_p(\d+)/) || [])[1];
const clipSec = (x) => Math.max(0, (x.size - 44) / 32000);
const queue = () => (T?.clips || []).filter((x) => !x.reviewed && x.kind !== "bg" && !skipped.has(x.kind + "/" + x.name))
  .sort((a, b) => ({ near: 0, pos: 1, false: 2 }[a.kind] - { near: 0, pos: 1, false: 2 }[b.kind]) || (new Date(b.time) - new Date(a.time)));

async function viewTraining() {
  try { T = await api("GET", "/api/train"); } catch (e) { toast(e.message, true); return; }
  T.clips ||= [];
  const used = (T.used_kb / 1024).toFixed(1);
  $("#view").innerHTML = `${secHead("training")}
  ${sec("Запись", "Включите на день-два и пользуйтесь колонкой как обычно: она запишет оклики, «почти оклики» и немного фона. Ложные срабатывания пишутся всегда",
    `<label class="sw"><input type="checkbox" id="t-mode"${T.mode ? " checked" : ""}><span class="tr"></span>Записывать для обучения</label>
    <div class="f"><label>Место на колонке под записи ${tip(T.cloud ? "На колонке лежат только свежие записи. Когда лимит заполнен, самые старые уходят на Google Диск пачкой по 10 — в «Проверке» они по-прежнему есть." : "Записи лежат на колонке, а места там мало. Когда лимит заполнен, сначала удаляются старый фон и «почти оклики». Подключите Google Диск — и старые записи будут уходить туда, а не удаляться.")}<b id="t-lim-v" class="val">${used} из ${T.limit_mb} МБ</b></label><input type="range" id="t-lim" min="1" max="20" value="${T.limit_mb}"><div class="scale"><span>1 МБ</span><span>свободно на колонке ${T.free_mb} МБ</span></div></div>
    <p class="note cloud-note">${T.cloud ? `${icon("cloud")}<span>На Google Диске <b>${T.remote || 0}</b> ${plural(T.remote || 0, "запись", "записи", "записей")} — папка «Jarvis — обучение». Старые уходят туда пачками по 10${T.offloading ? " · <b>отправляю…</b>" : ""}</span>` : `${icon("cloud")}<span>Подключите Google Диск в разделе «Фон комнаты» ниже — тогда старые записи будут уходить туда, а не удаляться</span>`}</p>
    ${T.cloud_error ? `<p class="err">Google Диск: ${esc(T.cloud_error)}</p>` : ""}`)}
  ${sec("Проверка", "Запись играет по кругу — ответьте, было ли в ней слово «Джарвис». Клавиши: 1 — да, 2 — нет, пробел — пауза", `<div id="rv"></div>`)}
  ${sec("Фон комнаты", "Несколько часов звука вашей комнаты — ТВ, разговоры, музыка, — чтобы модель не путала их с «Джарвис». Каждая минута сразу уходит на Google Диск; минуты, где звали Джарвиса, не отправляются", `<div id="bg"><p class="note">…</p></div>`)}
  ${sec("Обучить модель", "Когда записи размечены — скачайте их и обучите модель на компьютере с видеокартой NVIDIA, около 5 минут", trainStepsHtml())}
  ${sec("Все записи", "", `<div id="clips"></div>`)}`;
  drawReview(); drawBg(); drawClips();
  $("#t-mode").addEventListener("change", (e) => api("PUT", "/api/settings", { train_mode: e.target.checked }).then(() => { toast(e.target.checked ? "Запись для обучения включена" : "Запись выключена"); viewTraining(); }).catch((err) => toast(err.message, true)));
  $("#t-lim").addEventListener("input", (e) => ($("#t-lim-v").textContent = `${used} из ${e.target.value} МБ`));
  $("#t-lim").addEventListener("change", (e) => api("PUT", "/api/settings", { train_limit_mb: +e.target.value }).catch((err) => toast(err.message, true)));
  $("#t-model").addEventListener("change", (e) => { const file = e.target.files[0]; if (!file) return; act(null, async () => { await upload("PUT", "/api/train/model", file); await waitRestart("Модель загружена — Джарвис перезапускается"); }); });
  $("#t-restore")?.addEventListener("click", (e) => act(e.target, async () => { await api("POST", "/api/train/model/restore"); await waitRestart("Возвращаю прежнюю модель — Джарвис перезапускается"); }));
}

function trainStepsHtml() {
  const c = T.counts || {};
  return `<div class="row"><div class="chips"><span class="chip k-pos">Джарвис <b id="c-pos">${c.pos || 0}</b></span><span class="chip k-false">Не Джарвис <b id="c-false">${c.false || 0}</b></span><span class="chip">Фон <b>${c.bg || 0}</b></span></div>
      <span class="spacer"></span>${T.clips.length ? `<a class="btn primary" href="/api/train/download" download data-tip="Все записи — и с колонки, и с Google Диска — плюс текущая модель и скрипт finetune.py: распакуйте и запустите «python finetune.py»">Скачать записи</a>` : `<button class="btn" disabled>Скачать записи</button>`}</div>
    <ol class="steps">
      <li><b>Скачайте записи</b> — в архиве они уже разложены по вашим ответам. Фон с Google Диска (папка «Jarvis — фон комнаты») положите в <code>kws/bg_long</code>.</li>
      <li><b>Обучите</b> в папке проекта: <code>python kws/from_speaker.py jarvis-train-….zip</code> — около 3 минут на видеокарте, модель появится в <code>kws/out_next/wake.bin</code>.</li>
      <li><b>Загрузите</b> её сюда — колонка проверит файл и перезапустится. Стало хуже — верните прежнюю.</li></ol>
    <div class="row"><label class="btn" for="t-model">Загрузить wake.bin</label><input type="file" id="t-model" accept=".bin" class="hidden">${T.prev_model ? `<button class="btn ghost" id="t-restore">Вернуть прежнюю модель</button>` : ""}</div>`;
}

// ---- проверка: плеер, который всё время играет текущую запись по кругу
const RV = { audio: null, started: false, peaks: [], key: "" };
function rvAudio() {
  if (!RV.audio) {
    RV.audio = new Audio(); RV.audio.loop = true; RV.audio.preload = "auto";
    RV.audio.addEventListener("play", () => { RV.started = true; $("#rv-toggle") && ($("#rv-toggle").innerHTML = icon("pause", "solid")); Motion.add(rvAnim); });
    RV.audio.addEventListener("pause", () => { $("#rv-toggle") && ($("#rv-toggle").innerHTML = icon("play", "solid")); });
  }
  return RV.audio;
}
const rvAnim = {
  render() {
    const a = RV.audio, wave = $("#rv-wave");
    if (!a || !wave) return;
    const d = a.duration || 0, p = d ? a.currentTime / d : 0;
    const bars = wave.children, lit = Math.round(p * bars.length);
    for (let i = 0; i < bars.length; i++) bars[i].classList.toggle("on", i < lit);
    $("#rv-time").textContent = `${(a.currentTime || 0).toFixed(1)} / ${d ? d.toFixed(1) : "…"} с`;
  },
  frame() {},
  done() { return !RV.audio || RV.audio.paused || !$("#rv-wave"); },
};

async function rvWave(url) {
  const wave = $("#rv-wave");
  try {
    const buf = await (await fetch(url, { credentials: "same-origin" })).arrayBuffer();
    const ctx = rvWave.ctx ||= new (window.AudioContext || window.webkitAudioContext)();
    const data = (await ctx.decodeAudioData(buf)).getChannelData(0);
    const n = 72, step = Math.floor(data.length / n), peaks = [];
    for (let i = 0; i < n; i++) { let m = 0; for (let j = i * step; j < (i + 1) * step; j++) m = Math.max(m, Math.abs(data[j] || 0)); peaks.push(m); }
    const top = Math.max(0.05, ...peaks);
    if (!wave || $("#rv-wave") !== wave) return;
    [...wave.children].forEach((b, i) => b.style.setProperty("--h", `${Math.max(6, (peaks[i] / top) * 100).toFixed(0)}%`));
  } catch { /* волна — украшение */ }
}

function drawReview() {
  const q = queue(), x = q[0], box = $("#rv");
  if (!box) return;
  if (!x) {
    RV.audio?.pause();
    box.innerHTML = `<div class="rv-done">${icon("done")}<div><b>Всё проверено</b><small>${T.mode ? "Новые записи появятся здесь сами — заглядывайте раз в день-два." : "Включите запись вверху — колонка начнёт собирать оклики для проверки."}</small></div></div>`;
    return;
  }
  const p = clipProb(x), key = x.kind + "/" + x.name;
  box.innerHTML = `<div class="rvp">
    <div class="rvp-head"><span class="chip k-${x.kind}">${esc(GUESS[x.kind])}${p ? ` · ${p}%` : ""}</span><span class="spacer"></span><span class="muted">${esc(clipWhen(x))} · осталось ${q.length}</span></div>
    <div class="rvp-player">
      <button class="np-play" id="rv-toggle" type="button" aria-label="Пауза">${icon(RV.started ? "pause" : "play", "solid")}</button>
      <div class="rvp-wave" id="rv-wave" title="Нажмите — сыграть с этого места">${"<i></i>".repeat(72)}</div>
      <span class="rvp-time" id="rv-time">0.0 / … с</span>
    </div>
    <div class="rvp-q"><b>Было слово «Джарвис»?</b></div>
    <div class="rv-ans"><button class="btn primary" data-a="yes">Да, было <kbd>1</kbd></button><button class="btn" data-a="no">Нет <kbd>2</kbd></button></div>
    <div class="rv-alt"><button class="btn sm ghost" data-a="skip">Пропустить</button><button class="btn sm ghost danger" data-a="del">Не уверен — удалить</button></div>
  </div>`;
  const a = rvAudio();
  if (RV.key !== key) { RV.key = key; a.src = clipUrl(x); }
  // следующую запись (если она на Google Диске) колонка докачивает заранее — ответ «да/нет» без паузы
  const nx = q[1];
  if (nx?.remote && RV.pre !== nx.name) { RV.pre = nx.name; fetch(clipUrl(nx), { credentials: "same-origin" }).catch(() => {}); }
  rvWave(clipUrl(x));
  if (RV.started) a.play().catch(() => {});
  rvAnim.render();
  box.onclick = (e) => {
    const b = e.target.closest("[data-a]");
    if (b) return answer(b.dataset.a, b);
    if (e.target.closest("#rv-toggle")) return a.paused ? a.play().catch(() => {}) : a.pause();
    const w = e.target.closest("#rv-wave");
    if (w && a.duration) { const r = w.getBoundingClientRect(); a.currentTime = ((e.clientX - r.left) / r.width) * a.duration; a.play().catch(() => {}); }
  };
}

async function answer(a, btn) {
  const x = queue()[0];
  if (!x) return;
  RV.started = true;   // ответили — дальше записи сами играют по кругу
  if (a === "skip") { skipped.add(x.kind + "/" + x.name); drawReview(); return; }
  await act(btn, async () => {
    await api("POST", "/api/train/move", { kind: x.kind, name: x.name, to: a === "yes" ? "pos" : a === "no" ? "false" : "delete", reviewed: true });
    T = await api("GET", "/api/train"); T.clips ||= [];
  });
  const c = T.counts || {};
  if ($("#c-pos")) { $("#c-pos").textContent = c.pos || 0; $("#c-false").textContent = c.false || 0; }
  drawReview(); drawClips();
}

document.addEventListener("keydown", (e) => {
  if (page !== "training" || !$("#rv-toggle") || e.target.closest("input, textarea, select")) return;
  if (e.key === "1") answer("yes");
  else if (e.key === "2") answer("no");
  else if (e.key === " ") { e.preventDefault(); const a = rvAudio(); a.paused ? a.play().catch(() => {}) : a.pause(); }
});

// ---- все записи: аккуратный список, свой маленький плеер на строку
const LIST = { audio: null, key: "" };
function drawClips() {
  const box = $("#clips");
  if (!box) return;
  const c = T.counts || {}, all = T.clips;
  const shown = all.filter((x) => clipFilter === "all" || (clipFilter === "todo" ? !x.reviewed && x.kind !== "bg" : x.kind === clipFilter));
  const tabs = [["all", "Все", all.length], ["pos", "Джарвис", c.pos || 0], ["false", "Не Джарвис", c.false || 0], ["bg", "Фон", c.bg || 0], ["todo", "Не проверено", all.filter((x) => !x.reviewed && x.kind !== "bg").length]];
  box.innerHTML = `<div class="row"><div class="seg" id="cl-f">${tabs.map(([k, n, cnt]) => `<button type="button" data-f="${k}" class="${clipFilter === k ? "on" : ""}">${n} <small>${cnt}</small></button>`).join("")}</div>
      <span class="spacer"></span>${all.length ? `<button class="btn sm ghost danger" id="t-clear">Удалить все</button>` : ""}</div>
    <div class="clips">${shown.slice(0, clipLimit).map((x) => {
      const k = x.kind + "/" + x.name, playing = LIST.key === k && LIST.audio && !LIST.audio.paused;
      return `<div class="clip${playing ? " on" : ""}" data-key="${esc(k)}">
        <button class="ib clip-play" data-play="${esc(k)}" aria-label="Прослушать">${icon(playing ? "pause" : "play", "solid")}</button>
        <span class="clip-meta"><b>${esc(clipWhen(x))}${x.remote ? ` <span class="on-cloud" data-tip="На Google Диске — колонка докачает при прослушивании">${icon("cloud")}</span>` : ""}</b><small>${clipSec(x).toFixed(1)} с${clipProb(x) ? ` · ${clipProb(x)}%` : ""}${x.reviewed ? " · проверено" : ""}</small></span>
        <div class="seg sm">${x.kind === "bg" ? `<button type="button" class="on" disabled>Фон</button>` : `<button type="button" data-mv="pos" data-k="${x.kind}" data-n="${esc(x.name)}" class="${x.kind === "pos" ? "on" : ""}">Джарвис</button><button type="button" data-mv="false" data-k="${x.kind}" data-n="${esc(x.name)}" class="${x.kind === "false" ? "on" : ""}">Не Джарвис</button>`}</div>
        <button class="ib clip-del" data-mv="delete" data-k="${x.kind}" data-n="${esc(x.name)}" aria-label="Удалить">${icon("x")}</button>
      </div>`; }).join("") || `<p class="note">Здесь пока пусто</p>`}</div>
    ${shown.length > clipLimit ? `<button class="btn sm ghost" id="cl-more">Показать ещё ${Math.min(40, shown.length - clipLimit)} из ${shown.length - clipLimit}</button>` : ""}`;
  box.onclick = async (e) => {
    const f = e.target.closest("[data-f]"), mv = e.target.closest("[data-mv]"), pl = e.target.closest("[data-play]");
    if (f) { clipFilter = f.dataset.f; clipLimit = 40; drawClips(); return; }
    if (e.target.closest("#cl-more")) { clipLimit += 40; drawClips(); return; }
    if (e.target.closest("#t-clear")) { if (confirm("Удалить все записи для обучения? Сначала лучше скачать архив.")) act(null, async () => { await api("DELETE", "/api/train"); viewTraining(); }, "Записи удалены"); return; }
    if (pl) {
      const a = LIST.audio ||= new Audio();
      if (LIST.key === pl.dataset.play && !a.paused) a.pause();
      else { const [kind, ...n] = pl.dataset.play.split("/"); if (LIST.key !== pl.dataset.play) a.src = clipUrl({ kind, name: n.join("/") }); LIST.key = pl.dataset.play; a.play().catch(() => {}); }
      a.onended = a.onpause = a.onplay = () => drawClips();
      return;
    }
    if (mv && !mv.classList.contains("on")) await act(mv, async () => {
      await api("POST", "/api/train/move", { kind: mv.dataset.k, name: mv.dataset.n, to: mv.dataset.mv, reviewed: true });
      T = await api("GET", "/api/train"); T.clips ||= []; drawClips(); drawReview();
    });
  };
}

// ---------------------------------------------------------------- подсветка: выбор цвета
const LED_SWATCHES = ["#ffffff", "#4fd1c5", "#00b4ff", "#3d6bff", "#5a2dff", "#b04dff", "#ff4d8d", "#ff5a36", "#ffb020", "#3ddc84"];
const LED_STATES = [["listening", "Слушает", "кольцо плавно загорается и дышит"], ["thinking", "Думает", "огонёк бежит по кругу"], ["speaking", "Говорит", "вспыхивает в такт голосу"]];

// строка выбора цвета: живое мини-кольцо + образцы + свой цвет
function ledPickerHtml(led, prefix) {
  return `<div class="ledrows">${LED_STATES.map(([k, n, hint]) => `
    <div class="ledrow" data-k="${k}">
      <span class="ledring ${k}" data-ring="${prefix}-${k}"></span>
      <div class="ledname"><b>${n}</b><small>${hint}</small></div>
      <div class="swatches">${LED_SWATCHES.map((c) => `<button type="button" class="swatch${c === led[k] ? " on" : ""}" data-c="${c}" data-k="${k}" aria-label="Цвет ${c}"></button>`).join("")}
        <label class="swatch custom${LED_SWATCHES.includes(led[k]) ? "" : " on"}" aria-label="Свой цвет"><input type="color" data-k="${k}" value="${esc(led[k])}"></label></div>
    </div>`).join("")}</div>`;
}

// раскрасить образцы и кольца (цвета — через CSSOM: встроенные style="" запрещены политикой страницы)
function paintLeds(root, led, prefix) {
  $$(".swatch[data-c]", root).forEach((b) => b.style.setProperty("--c", b.dataset.c));
  LED_STATES.forEach(([k]) => {
    const ring = $(`[data-ring="${prefix}-${k}"]`, root);
    if (ring) { ring.style.setProperty("--c", led[k]); ring.style.setProperty("--b", String(0.35 + 0.65 * (led.brightness / 100))); }
    const custom = $(`.ledrow[data-k="${k}"] .swatch.custom`, root);
    if (custom) custom.style.setProperty("--c", led[k]);
  });
}

// обработчики: onPick(k, color) — выбран цвет; на колонке он сразу загорается на 3 секунды
function ledPickerBind(root, led, prefix, onPick) {
  paintLeds(root, led, prefix);
  const pick = (k, c) => {
    led[k] = c;
    $$(`.ledrow[data-k="${k}"] .swatch`, root).forEach((s) => s.classList.toggle("on", s.dataset.c === c || (s.classList.contains("custom") && !LED_SWATCHES.includes(c))));
    paintLeds(root, led, prefix);
    onPick(k, c);
    if (S.speaker.leds) api("POST", "/api/leds/preview", { mode: k, color: c, brightness: led.brightness }).catch(() => {});
  };
  root.addEventListener("click", (e) => { const s = e.target.closest(".swatch[data-c]"); if (s) pick(s.dataset.k, s.dataset.c); });
  root.addEventListener("input", (e) => { if (e.target.matches('input[type=color]')) { led[e.target.dataset.k] = e.target.value; paintLeds(root, led, prefix); } });
  root.addEventListener("change", (e) => { if (e.target.matches('input[type=color]')) pick(e.target.dataset.k, e.target.value); });
}

// ---------------------------------------------------------------- плитки на главной (как в yapanel) и графики
const fmtN = (v, dec = 0) => (v == null || isNaN(v) ? "—" : Number(v).toLocaleString("ru-RU", { minimumFractionDigits: dec, maximumFractionDigits: dec }).replace("-", "−"));
const KPI = [
  { id: "temp", label: "Температура" },
  { id: "mem", label: "Память", tip: "Сколько оперативной памяти колонки занято. Джарвис занимает около 25 МБ." },
  { id: "cpu", label: "Процессор" },
  { id: "wifi", label: "Wi-Fi", tip: "Уровень сигнала: от −30 до −60 дБм — отлично, до −70 — нормально, ниже — связь может рваться, и Джарвис будет отвечать с задержкой." },
];
const kpiHtml = (k) => `<div class="kpi" id="k-${k.id}"><span class="label">${k.label}${k.tip ? " " + tip(k.tip) : ""}</span><div class="value"><b>—</b><span></span></div><div class="kbar"><i></i></div><span class="meta"></span>
  <svg class="spark" viewBox="0 0 200 46" aria-hidden="true"><path class="a"/><path class="l"/></svg></div>`;
let ST = { pts: [], at: 0 };

async function statsData() {
  ST.at = Date.now();
  try { ST.pts = (await api("GET", "/api/stats?range=900")).points || []; } catch { return; }
  drawKpis();
}

// число «набегает» к новому значению (при первом показе — от нуля)
function countTo(el, v, dec = 0) {
  if (v == null) { el.textContent = "—"; delete el.dataset.v; return; }
  const first = el.dataset.v == null, from = first ? 0 : +el.dataset.v;
  if (from === v) { if (first) { el.dataset.v = v; el.textContent = fmtN(v, dec); } return; }
  el.dataset.v = v;
  if (Motion.reduce) { el.textContent = fmtN(v, dec); return; }
  const t0 = performance.now(), d = 800;
  const step = (t) => {
    if (+el.dataset.v !== v) return;
    const k = Math.min(1, (t - t0) / d), e = 1 - Math.pow(1 - k, 3);
    el.textContent = fmtN(from + (v - from) * e, dec);
    if (k < 1) requestAnimationFrame(step);
  };
  requestAnimationFrame(step);
}

function sparkPaths(vals, w = 100, h = 40) {
  const v = vals.filter((x) => x != null && !isNaN(x));
  if (v.length < 2) return null;
  let lo = Math.min(...v), hi = Math.max(...v);
  if (hi - lo < 1e-6) { hi += 1; lo -= 1; }
  const top = h * 0.3;
  const pts = v.map((y, i) => [(i / (v.length - 1)) * w, top + (1 - (y - lo) / (hi - lo)) * (h - top - 2)]);
  const l = pts.map((p, i) => (i ? "L" : "M") + p[0].toFixed(2) + " " + p[1].toFixed(2)).join("");
  return { l, a: l + `L${w} ${h}L0 ${h}Z` };
}

function setKpi(id, val, unit, frac, meta, series, dec = 0, level = "") {
  const box = $("#k-" + id);
  if (!box) return;
  countTo($(".value b", box), val, dec);
  $(".value span", box).textContent = val == null ? "" : unit;
  const bar = $(".kbar", box);
  bar.className = "kbar" + (level ? " " + level : "");
  $(".kbar i", box).style.setProperty("width", `${Math.round(Motion.clamp(frac || 0, 0, 1) * 100)}%`);
  $(".meta", box).textContent = meta || "";
  const svg = $(".spark", box), w = Math.round(svg.clientWidth) || 200, h = 46, p = sparkPaths(series, w, h);
  if (!p) return;
  svg.setAttribute("viewBox", `0 0 ${w} ${h}`);
  $("path.l", svg).setAttribute("d", p.l);
  $("path.a", svg).setAttribute("d", p.a);
  if (!svg.dataset.drawn) { svg.dataset.drawn = "1"; drawIn($("path.l", svg), $("path.a", svg), 1300); }
}

function drawKpis() {
  if (!$("#kpis")) return;
  const sys = S.sys || {}, last = ST.pts[ST.pts.length - 1] || {};
  const used = (sys.mem_total_mb || 0) - (sys.mem_avail_mb || 0);
  setKpi("temp", sys.temp_c ? Math.round(sys.temp_c) : null, "°C", (sys.temp_c || 0) / 85, "процессор", ST.pts.map((p) => p.temp || null), 0, sys.temp_c > 75 ? "bad" : sys.temp_c > 65 ? "warn" : "");
  setKpi("mem", sys.mem_total_mb ? used : null, sys.mem_total_mb ? `из ${sys.mem_total_mb} МБ` : "", used / (sys.mem_total_mb || 1), sys.mem_total_mb ? `свободно ${sys.mem_avail_mb} МБ` : "", ST.pts.map((p) => p.mem));
  const cpu = ST.pts.length > 1 ? Math.round(last.cpu) : null;
  const me = ST.pts.length > 1 ? Math.round(last.me || 0) : null;
  setKpi("cpu", cpu, "%", (cpu || 0) / 100, me != null ? `из них Джарвис — ${me}%` : "все ядра колонки", ST.pts.slice(1).map((p) => p.cpu), 0, cpu > 85 ? "warn" : "");
  const w = sys.wifi_dbm || null, q = w ? Motion.clamp((w + 90) / 60, 0, 1) : 0;
  setKpi("wifi", w, "дБм", q, !w ? "" : w >= -60 ? "сигнал отличный" : w >= -70 ? "сигнал нормальный" : "сигнал слабый — ответы могут задерживаться", ST.pts.map((p) => p.wifi || null), 0, w && w < -70 ? "warn" : "");
}

const CHARTS = [
  { k: "cpu", t: "Процессор", u: "%", sub: "загрузка всех ядер", fix: [0, 100] },
  { k: "temp", t: "Температура", u: "°C", sub: "процессор", dec: 1 },
  { k: "mem", t: "Память занята", u: "МБ", sub: () => (S.sys.mem_total_mb ? `из ${S.sys.mem_total_mb} МБ` : "") },
  { k: "wifi", t: "Сигнал Wi-Fi", u: "дБм", sub: "ближе к нулю — лучше" },
  { k: "disk", t: "Свободно на /data", u: "МБ", sub: "меньше 5 МБ — обновления могут не встать" },
  { k: "me", t: "Сам Джарвис", u: "%", sub: "сколько процессора колонки занимает Джарвис", fix: [0, 100] },
];
const CH = { range: 3600, pts: [], at: 0 };

function viewCharts() {
  $("#view").innerHTML = `<header class="ph ph-row"><div><h1>Графики</h1><p class="sub">Раз в 15 секунд за последние сутки. История живёт в памяти колонки и начинается заново после перезапуска Джарвиса</p></div>
    <div class="seg" id="ch-range">${[[900, "15 мин"], [3600, "1 ч"], [21600, "6 ч"], [86400, "24 ч"]].map(([s, n]) => `<button type="button" data-r="${s}" class="${s === CH.range ? "on" : ""}">${n}</button>`).join("")}</div></header>
  <div class="charts stagger">${CHARTS.map((c) => `<section class="card"><div class="chart-h"><div><h2>${c.t}</h2><small id="cs-${c.k}"></small></div><div class="value"><b id="cv-${c.k}">—</b><span>${c.u}</span></div></div><div class="chart" id="ch-${c.k}"></div></section>`).join("")}</div>`;
  $$("#ch-range [data-r]").forEach((b) => b.addEventListener("click", () => {
    CH.range = +b.dataset.r;
    $$("#ch-range button").forEach((x) => x.classList.toggle("on", x === b));
    chartsData(true);
  }));
  chartsData(true);
}

async function chartsData(fresh) {
  if (page !== "charts") return;
  CH.at = Date.now();
  try { CH.pts = (await api("GET", "/api/stats?range=" + CH.range)).points || []; } catch { return; }
  for (const c of CHARTS) drawChart(c, fresh);
}

function drawChart(c, fresh) {
  const box = $("#ch-" + c.k);
  if (!box) return;
  const pts = CH.pts.filter((p, i) => p[c.k] != null && !((c.k === "wifi" || c.k === "temp") && !p[c.k]) && !((c.k === "cpu" || c.k === "me") && i === 0));
  const last = pts[pts.length - 1];
  const val = $("#cv-" + c.k);
  if (last) countTo(val, +last[c.k].toFixed(c.dec || 0), c.dec || 0); else val.textContent = "—";
  $("#cs-" + c.k).textContent = typeof c.sub === "function" ? c.sub() : c.sub;
  if (pts.length < 2) { box.innerHTML = `<p class="empty">Данные копятся — первые точки появятся через минуту</p>`; return; }
  const W = Math.max(240, box.clientWidth), H = 150, L = 40, B = 20, T = 6, R = 6;
  const vs = pts.map((p) => p[c.k]);
  let lo = Math.min(...vs), hi = Math.max(...vs);
  if (c.fix) [lo, hi] = c.fix;
  else { const pad = (hi - lo) * 0.2 || Math.max(1, Math.abs(hi) * 0.05); lo -= pad; hi += pad; }
  const t0 = pts[0].t, t1 = Math.max(pts[pts.length - 1].t, t0 + 1);
  const X = (t) => L + ((t - t0) / (t1 - t0)) * (W - L - R), Y = (v) => T + (1 - (v - lo) / (hi - lo)) * (H - T - B);
  const line = pts.map((p, i) => (i ? "L" : "M") + X(p.t).toFixed(1) + " " + Y(p[c.k]).toFixed(1)).join("");
  const area = line + `L${X(t1).toFixed(1)} ${H - B}L${L} ${H - B}Z`;
  const dec = hi - lo < 5 ? 1 : 0;
  const yt = [lo, (lo + hi) / 2, hi].map((v) => `<line x1="${L}" x2="${W - R}" y1="${Y(v).toFixed(1)}" y2="${Y(v).toFixed(1)}"/>`).join("");
  const yl = [lo, (lo + hi) / 2, hi].map((v) => `<text class="ax" x="${L - 8}" y="${(Y(v) + 3.5).toFixed(1)}" text-anchor="end">${fmtN(v, dec)}</text>`).join("");
  const hhmm = (t) => new Date(t * 1000).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
  const xl = [0.12, 0.38, 0.64, 0.9].map((f) => { const t = t0 + (t1 - t0) * f; return `<text class="ax" x="${X(t).toFixed(1)}" y="${H - 4}" text-anchor="middle">${hhmm(t)}</text>`; }).join("");
  box.innerHTML = `<svg viewBox="0 0 ${W} ${H}" aria-hidden="true"><g class="grid">${yt}</g>${yl}${xl}<path class="a" d="${area}"/><path class="l" d="${line}"/>
    <line class="cross" y1="${T}" y2="${H - B}" x1="-10" x2="-10"/><circle class="pt" r="4" cx="-10" cy="-10"/></svg><div class="ctip"></div>`;
  if (fresh) drawIn($("path.l", box), $("path.a", box));
  const svg = $("svg", box), cross = $(".cross", box), pt = $(".pt", box), tipEl = $(".ctip", box);
  box.onpointermove = (e) => {
    const r = svg.getBoundingClientRect(), x = ((e.clientX - r.left) / r.width) * W;
    const t = t0 + ((x - L) / (W - L - R)) * (t1 - t0);
    let best = pts[0];
    for (const p of pts) if (Math.abs(p.t - t) < Math.abs(best.t - t)) best = p;
    const bx = X(best.t), by = Y(best[c.k]);
    cross.setAttribute("x1", bx); cross.setAttribute("x2", bx); pt.setAttribute("cx", bx); pt.setAttribute("cy", by);
    tipEl.textContent = `${fmtN(best[c.k], c.dec || 0)} ${c.u} · ${hhmm(best.t)}`;
    tipEl.style.setProperty("left", `${(bx / W) * 100}%`);
  };
}
let chartsResize = 0;
window.addEventListener("resize", () => { clearTimeout(chartsResize); chartsResize = setTimeout(() => { if (page === "charts") for (const c of CHARTS) drawChart(c, false); if (page === "home") drawKpis(); }, 150); });

// край карточки светится там, где указатель
document.addEventListener("pointermove", (e) => {
  const c = e.target.closest?.(".card, .sec, .np, .kpi");
  if (!c) return;
  const r = c.getBoundingClientRect();
  c.style.setProperty("--mx", `${e.clientX - r.left}px`);
  c.style.setProperty("--my", `${e.clientY - r.top}px`);
}, { passive: true });

// плашка активного пункта меню переезжает на пружине
function navPill() {
  const nav = $("#nav");
  if (!nav) return;
  let pill = $(".nav-pill", nav);
  if (!pill) { pill = document.createElement("span"); pill.className = "nav-pill instant"; nav.prepend(pill); nav.classList.add("has-pill"); }
  const t = $(".nav > a.on, .prof.on", nav);
  if (!t || !t.offsetHeight) { pill.style.setProperty("opacity", "0"); return; }
  pill.style.setProperty("transform", `translateY(${t.offsetTop}px)`);
  pill.style.setProperty("height", `${t.offsetHeight}px`);
  pill.style.setProperty("opacity", "1");
  requestAnimationFrame(() => requestAnimationFrame(() => pill.classList.remove("instant")));
}

// ---------------------------------------------------------------- плеер — отдельная страница
const artOf = (t, big) => !t?.ID ? "" : M?.source === "yandex" ? (M.cover || "") : `https://i.ytimg.com/vi/${t.ID}/${big ? "maxresdefault" : "mqdefault"}.jpg`;
const musicLink = () => !M?.track?.ID ? "" : M.link || `https://youtu.be/${M.track.ID}`;

function viewPlayer() {
  $("#view").innerHTML = `<header class="ph"><h1>Плеер</h1><p class="sub">Всё, что играет на колонке: YouTube Джарвиса и Яндекс Музыка Алисы</p></header>
  <section class="card pl-hero" id="plh" data-state="idle"><img class="pl-glow" id="pl-glow" alt="">
    <div class="pl-top">
      <div class="pl-cover" id="pl-cover">${icon("music")}<img id="pl-img" alt=""></div>
      <div class="pl-meta">
        <div class="pl-kick"><span class="np-eq" aria-hidden="true"><i></i><i></i><i></i></span><span id="pl-kick">Тишина</span></div>
        <h2 class="pl-title mq-line" id="pl-title"></h2>
        <div class="pl-artist" id="pl-artist"></div>
        <div class="pl-chips" id="pl-chips"></div>
      </div>
    </div>
    <div class="pl-bottom">
      <div class="progress pl-seek"><span class="roll" id="pl-cur"></span><div id="pl-seek"></div><span id="pl-dur"></span></div>
      <div class="pl-bar">
        <div class="pl-l">
          <button class="cbtn" type="button" data-p="radio" id="pl-radio" data-tip="Радио: когда очередь кончится, играть похожее" aria-label="Радио">${icon("radio")}</button>
          <button class="cbtn" type="button" data-p="copy" id="pl-copy" data-tip="Скопировать ссылку" aria-label="Скопировать ссылку">${icon("link")}</button>
          <a class="cbtn" id="pl-open" target="_blank" rel="noopener" data-tip="Открыть на YouTube" aria-label="Открыть в браузере">${icon("ext")}</a>
        </div>
        <div class="pl-controls">
          <button class="cbtn" type="button" data-p="stop" data-tip="Выключить" aria-label="Выключить">${icon("stop", "solid")}</button>
          <button class="cbtn" type="button" data-p="prev" id="pl-prev" aria-label="Предыдущий">${icon("prev", "solid")}</button>
          <button class="cbtn big" type="button" data-p="toggle" id="pl-toggle" aria-label="Пауза">${icon("play", "solid")}</button>
          <button class="cbtn" type="button" data-p="next" id="pl-next" aria-label="Следующий">${icon("skip", "solid")}</button>
        </div>
        <div class="pl-vol" id="np-vol" title="Громкость колонки">${icon("speaker")}<div id="vol"></div><b class="vol-big" id="vol-big">–</b></div>
      </div>
    </div>
  </section>
  <div class="cols stagger">
    <section class="card"><div class="card-head"><h2>Очередь</h2><span class="meta" id="pl-qmeta"></span></div>
      <form class="q-add" id="pl-form"><label class="q-field">${icon("search")}<input type="text" id="pl-q" placeholder="Название или ссылка на YouTube" autocomplete="off" aria-label="Что включить"></label>
        <button class="cbtn" type="submit" data-a="play" data-tip="Включить сейчас" aria-label="Включить сейчас">${icon("play", "solid")}</button>
        <button class="cbtn" type="submit" data-a="queue" data-tip="В очередь" aria-label="В очередь">${icon("plus")}</button></form>
      <div class="qlist" id="pl-queue"></div></section>
    <section class="card"><div class="card-head"><h2>Недавно играло</h2><span class="meta">YouTube · нажмите, чтобы включить</span></div><div class="qlist" id="pl-hist"></div></section>
  </div>`;
  $("#plh").addEventListener("click", (e) => {
    const b = e.target.closest("[data-p]");
    if (!b || b.disabled) return;
    musicAct(b.dataset.p, b);
  });
  $("#pl-form").addEventListener("submit", (e) => {
    e.preventDefault();
    const q = $("#pl-q").value.trim(), a = e.submitter?.dataset.a || "play";
    if (!q) return;
    $("#pl-q").value = "";
    if (a === "play") { M = { ...(M || {}), loading: true }; drawPlayerPage(); }
    music({ action: a, query: q }).then((ok) => ok && a === "queue" && toast("Добавлено в очередь"));
  });
  const onRow = (e) => {
    const rm = e.target.closest("[data-un]");
    if (rm) { e.stopPropagation(); music({ action: "unqueue", index: +rm.dataset.un }); return; }
    const r = e.target.closest(".qrow[data-id]");
    if (!r || r.classList.contains("cur")) return;
    if (r.dataset.qi != null) music({ action: "unqueue", index: +r.dataset.qi }).then(() => music({ action: "id", query: r.dataset.id }));
    else music({ action: "id", query: r.dataset.id });
  };
  $("#pl-queue").addEventListener("click", onRow);
  $("#pl-hist").addEventListener("click", onRow);
  makeSliders($("#pl-seek"), $("#vol"));
  drawVolume();
  if (M) drawPlayerPage();
  musicData();
}

function swapText(el, text) {
  if (el.title === text && el.firstChild) return;
  el.title = text;
  if (el.classList.contains("mq-line")) {   // не влезает — бежит туда и обратно (как в yapanel)
    el.innerHTML = `<span class="mq">${esc(text)}</span>`;
    el.classList.remove("run");
    requestAnimationFrame(() => {
      const over = el.scrollWidth - el.clientWidth;
      if (over > 4) { el.style.setProperty("--mq", `${-over - 24}px`); el.style.setProperty("--mqt", `${Math.max(6, over / 28)}s`); el.classList.add("run"); }
    });
  } else el.textContent = text;
  el.classList.remove("swap"); void el.offsetWidth; el.classList.add("swap");
}

function qrowHtml(t, { badge = "", cur = false, qi = null, art = "" } = {}) {
  const img = art || (t.ID && !/^\d+$/.test(t.ID) ? `https://i.ytimg.com/vi/${esc(t.ID)}/mqdefault.jpg` : "");
  return `<div class="qrow${cur ? " cur" : ""}" data-id="${esc(t.ID)}"${qi != null ? ` data-qi="${qi}"` : ""}${cur ? "" : ` role="button" tabindex="0"`}>
    ${img ? `<img src="${esc(img)}" alt="" loading="lazy">` : `<span class="qph"></span>`}<span class="tt"><b>${esc(t.Title || "…")}</b><span>${esc(t.Author || "")}</span></span>${badge}
    ${qi != null ? `<button class="ib" type="button" data-un="${qi}" aria-label="Убрать из очереди" data-tip="Убрать">${icon("x")}</button>` : ""}</div>`;
}

function drawPlayerPage() {
  const h = $("#plh");
  if (!h || !M) return;
  const t = M.track || {}, on = M.playing || M.paused, ya = M.source === "yandex";
  h.dataset.state = M.loading && !on ? "loading" : !on ? "idle" : M.paused ? "paused" : "playing";
  const art = on ? artOf(t, true) : "", glow = $("#pl-glow");
  setArt($("#pl-img"), art, t);
  const g = art.replace("maxresdefault", "mqdefault");
  if (glow.dataset.src !== g) { glow.dataset.src = g; if (g) glow.src = g; else glow.removeAttribute("src"); }
  swapText($("#pl-title"), on ? t.Title || "…" : M.loading ? "Ищу и включаю…" : "Ничего не играет");
  swapText($("#pl-artist"), on ? t.Author || (ya ? "Яндекс Музыка" : "YouTube") : "Скажите «Джарвис, включи с ютуба …» или попросите Алису");
  $("#pl-kick").textContent = on ? `${ya ? "Яндекс Музыка" : "YouTube"}${M.paused ? " · пауза" : ""}` : M.loading ? "YouTube · загрузка" : "Тишина";
  const chips = on ? [ya ? "через Алису" : "через Джарвиса", M.album && `Альбом «${M.album}»`, !ya && M.radio && "радио"].filter(Boolean) : [];
  const link = on ? musicLink() : "", open = $("#pl-open");
  if (link) open.href = link; else open.removeAttribute("href");
  open.classList.toggle("hidden", !link);
  open.dataset.tip = ya ? "Открыть в Яндекс Музыке" : "Открыть на YouTube";
  const ch = chips.map((c) => `<span class="chip">${esc(c)}</span>`).join("");
  if ($("#pl-chips").dataset.h !== ch) { $("#pl-chips").dataset.h = ch; $("#pl-chips").innerHTML = ch; }
  $("#pl-dur").textContent = M.dur ? clockStr(M.dur) : "";
  $("#pl-toggle").innerHTML = icon(M.paused || !on ? "play" : "pause", "solid");
  $("#pl-toggle").disabled = !on;
  $("#pl-prev").disabled = !on || (!ya && (M.history || []).length < 2);
  $("#pl-next").disabled = !on || (!ya && !(M.next || []).length);
  $("#pl-radio").classList.toggle("on", !!M.radio);
  $("#pl-radio").classList.toggle("hidden", ya);
  $("#pl-copy").classList.toggle("hidden", !link);
  let qh;
  if (ya) qh = on ? qrowHtml(t, { cur: true, badge: `<span class="np-eq q-eq" aria-hidden="true"><i></i><i></i><i></i></span>`, art: M.cover }) + `<p class="empty">Очередью Яндекс Музыки управляет Алиса: «Алиса, следующий трек», «Алиса, включи похожее».</p>` : "";
  else {
    const next = M.next || [], nq = M.queue || 0;
    qh = (on ? qrowHtml(t, { cur: true, badge: `<span class="np-eq q-eq" aria-hidden="true"><i></i><i></i><i></i></span>` }) : "") +
      next.map((x, i) => qrowHtml(x, { qi: i < nq ? i : null, badge: i < nq ? `<span class="badge">${i + 1}</span>` : `<span class="badge">${M.radio ? "похожее" : "дальше"}</span>` })).join("");
  }
  if (!qh) qh = `<p class="empty">Очередь пуста. Найдите трек выше и нажмите «+» или скажите «Джарвис, добавь в очередь …»</p>`;
  if ($("#pl-queue").dataset.h !== qh) { $("#pl-queue").dataset.h = qh; $("#pl-queue").innerHTML = qh; }
  $("#pl-qmeta").textContent = ya ? "Яндекс Музыка" : `${M.queue || 0} в очереди${M.radio ? " · дальше похожее" : ""}`;
  const hist = (M.history || []).filter((x) => !(on && !ya && x.ID === t.ID)).slice(0, 12);
  const hh = hist.length ? hist.map((x) => qrowHtml(x)).join("") : `<p class="empty">Здесь появится то, что включали с YouTube</p>`;
  if ($("#pl-hist").dataset.h !== hh) { $("#pl-hist").dataset.h = hh; $("#pl-hist").innerHTML = hh; }
  Motion.add(npAnim);
}

// ---------------------------------------------------------------- линия графика «прорисовывается» (длина пути считается в пикселях)
function drawIn(line, area, ms = 1100) {
  if (Motion.reduce || !line.getTotalLength) return;
  const len = line.getTotalLength();
  line.style.setProperty("transition", "none");
  line.style.setProperty("stroke-dasharray", `${len} ${len}`);
  line.style.setProperty("stroke-dashoffset", String(len));
  if (area) { area.style.setProperty("transition", "none"); area.style.setProperty("opacity", "0"); }
  line.getBoundingClientRect();
  requestAnimationFrame(() => {
    line.style.setProperty("transition", `stroke-dashoffset ${ms}ms cubic-bezier(.22,1,.36,1)`);
    line.style.setProperty("stroke-dashoffset", "0");
    if (area) { area.style.setProperty("transition", `opacity ${ms * 0.7}ms ease ${ms * 0.35}ms`); area.style.setProperty("opacity", "1"); }
  });
  setTimeout(() => { if (line.isConnected) { line.style.removeProperty("stroke-dasharray"); line.style.removeProperty("stroke-dashoffset"); } }, ms + 120);
}

// ---------------------------------------------------------------- подсветка: «лаборатория цвета» — кольцо оттенков, HEX, насыщенность
const hsv2hex = (h, s, v = 1) => {
  const f = (n) => { const k = (n + h / 60) % 6; return v - v * s * Math.max(0, Math.min(k, 4 - k, 1)); };
  return "#" + [f(5), f(3), f(1)].map((x) => Math.round(x * 255).toString(16).padStart(2, "0")).join("");
};
const hex2hs = (hex) => {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
  if (!m) return null;
  const n = parseInt(m[1], 16), r = (n >> 16 & 255) / 255, g = (n >> 8 & 255) / 255, b = (n & 255) / 255;
  const mx = Math.max(r, g, b), mn = Math.min(r, g, b), d = mx - mn;
  let h = 0;
  if (d) h = mx === r ? ((g - b) / d) % 6 : mx === g ? (b - r) / d + 2 : (r - g) / d + 4;
  return { h: (h * 60 + 360) % 360, s: mx ? d / mx : 0 };
};
const LED_PRESETS = ["#ffffff", "#ff3b30", "#ff9500", "#ffcc00", "#34c759", "#00c7be", "#00b4ff", "#3d6bff", "#5a2dff", "#ff2d92"];

function ledLabHtml() {
  return `<div class="ledlab">
    <div class="hue-ring" id="lp-ring" role="slider" tabindex="0" aria-label="Оттенок"><div class="hue-knob" id="lp-knob"></div>
      <div class="hue-core" id="lp-core" data-fx="listening"><span class="core-glow"></span><span class="core-dot"></span></div></div>
    <div class="ledlab-side">
      <div class="seg" id="lp-fx"><button type="button" data-fx="listening" class="on">Дыхание</button><button type="button" data-fx="thinking">Огонёк по кругу</button><button type="button" data-fx="speaking">Пульс</button></div>
      <div class="hexrow"><span class="hexdot" id="lp-dot"></span><input type="text" id="lp-hex" value="#00B4FF" maxlength="7" spellcheck="false" autocomplete="off" aria-label="Цвет в HEX"><span class="meta">впишите цвет или крутите кольцо</span></div>
      <div class="f"><label>Насыщенность <b class="val" id="lp-sat-v"></b></label><div id="lp-sat"></div></div>
      <div class="f"><label>Яркость <b class="val" id="lp-br-v"></b></label><div id="lp-br"></div></div>
      <div class="presets" id="lp-pre">${LED_PRESETS.map((c) => `<button type="button" class="swatch" data-c="${c}" aria-label="Цвет ${c}"></button>`).join("")}</div>
      <div class="row"><button class="btn primary" id="lp-go">Зажечь на 30 секунд</button><button class="btn ghost" id="lp-off">Погасить</button><span class="meta" id="lp-live"></span></div>
    </div></div>`;
}

function ledLabBind() {
  const st = { h: 197, s: 1, br: 70, fx: "listening", lit: false };
  const ring = $("#lp-ring"), knob = $("#lp-knob"), core = $("#lp-core");
  let sendT = 0, lastSend = 0;
  const color = () => hsv2hex(st.h, st.s);
  const send = () => {
    lastSend = Date.now();
    st.lit = true;
    $("#lp-live").textContent = "горит на колонке";
    api("POST", "/api/leds/preview", { mode: st.fx, color: color(), brightness: st.br, seconds: 30 }).catch((e) => toast(e.message, true));
  };
  const sendSoon = () => { clearTimeout(sendT); const wait = Math.max(0, 140 - (Date.now() - lastSend)); sendT = setTimeout(send, wait); };
  const paint = (fromHex) => {
    const c = color();
    const a = ((st.h - 90) * Math.PI) / 180;
    knob.style.setProperty("left", `${50 + 40 * Math.cos(a)}%`);
    knob.style.setProperty("top", `${50 + 40 * Math.sin(a)}%`);
    knob.style.setProperty("--c", hsv2hex(st.h, 1));
    core.style.setProperty("--c", c);
    core.style.setProperty("--b", String(0.35 + 0.65 * (st.br / 100)));
    core.dataset.fx = st.fx;
    $("#lp-dot").style.setProperty("--c", c);
    if (!fromHex) $("#lp-hex").value = c.toUpperCase();
    $("#lp-sat-v").textContent = Math.round(st.s * 100) + "%";
    $("#lp-br-v").textContent = st.br + "%";
    $$("#lp-pre .swatch").forEach((b) => b.classList.toggle("on", b.dataset.c.toLowerCase() === c.toLowerCase()));
  };
  const hueAt = (e) => { const r = ring.getBoundingClientRect(); return (Math.atan2(e.clientY - (r.top + r.height / 2), e.clientX - (r.left + r.width / 2)) * 180 / Math.PI + 90 + 360) % 360; };
  let drag = false;
  ring.addEventListener("pointerdown", (e) => { if (e.target.closest(".hue-core")) return; drag = true; ring.setPointerCapture(e.pointerId); st.h = hueAt(e); paint(); sendSoon(); });
  ring.addEventListener("pointermove", (e) => { if (!drag) return; st.h = hueAt(e); paint(); sendSoon(); });
  ring.addEventListener("pointerup", () => { drag = false; });
  ring.addEventListener("keydown", (e) => { const d = { ArrowRight: 5, ArrowUp: 5, ArrowLeft: -5, ArrowDown: -5 }[e.key]; if (d == null) return; e.preventDefault(); st.h = (st.h + d + 360) % 360; paint(); sendSoon(); });
  core.addEventListener("click", send);
  $$("#lp-pre .swatch").forEach((b) => b.style.setProperty("--c", b.dataset.c));
  $("#lp-pre").addEventListener("click", (e) => { const b = e.target.closest("[data-c]"); if (!b) return; const v = hex2hs(b.dataset.c); st.h = v.h; st.s = v.s; satJ.set(Math.round(st.s * 100), true); paint(); send(); });
  $("#lp-hex").addEventListener("input", (e) => {
    let v = e.target.value.trim();
    if (!v.startsWith("#")) v = "#" + v;
    const hs = hex2hs(v);
    e.target.classList.toggle("bad", !hs && v.length === 7);
    if (!hs) return;
    st.h = hs.h; st.s = hs.s; satJ.set(Math.round(st.s * 100), true); paint(true); sendSoon();
  });
  const satJ = jelly($("#lp-sat"), { label: "Насыщенность", min: 0, max: 100, step: 1, value: 100, format: (v) => v + "%", onInput: (v) => { st.s = v / 100; paint(); sendSoon(); }, onChange: (v) => { st.s = v / 100; paint(); send(); } });
  jelly($("#lp-br"), { label: "Яркость", min: 5, max: 100, step: 5, value: st.br, format: (v) => v + "%", onInput: (v) => { st.br = v; paint(); sendSoon(); }, onChange: (v) => { st.br = v; paint(); send(); } });
  $$("#lp-fx button").forEach((b) => b.addEventListener("click", () => { $$("#lp-fx button").forEach((x) => x.classList.toggle("on", x === b)); st.fx = b.dataset.fx; paint(); send(); }));
  $("#lp-go").addEventListener("click", send);
  $("#lp-off").addEventListener("click", () => { st.lit = false; $("#lp-live").textContent = ""; api("POST", "/api/leds/preview", { mode: "off" }).catch(() => {}); });
  paint();
}

// ---------------------------------------------------------------- логи
const LOGS = { src: "jarvis", q: "", auto: true, at: 0, html: "" };
function viewLogs() {
  $("#view").innerHTML = `<header class="ph ph-row"><div><h1>Логи</h1><p class="sub">Что делали Джарвис и Алиса — последние 400 строк. Тексты разговоров пишутся только в журнал Джарвиса</p></div>
    <div class="actions"><div class="seg" id="lg-src"><button type="button" data-s="jarvis" class="${LOGS.src === "jarvis" ? "on" : ""}">Джарвис</button><button type="button" data-s="alice" class="${LOGS.src === "alice" ? "on" : ""}">Алиса</button><button type="button" data-s="media" class="${LOGS.src === "media" ? "on" : ""}">Звук и музыка</button></div></div></header>
  <section class="card logs-card">
    <div class="logs-bar"><label class="search">${icon("search")}<input type="search" id="lg-q" placeholder="Найти в логе" value="${esc(LOGS.q)}" autocomplete="off"></label>
      <label class="sw"><input type="checkbox" id="lg-auto"${LOGS.auto ? " checked" : ""}><span class="tr"></span>Обновлять сами</label>
      <button class="btn sm" id="lg-copy">${icon("link")}Скопировать</button></div>
    <div class="logbox" id="logbox"><p class="empty">Загружаю…</p></div>
  </section>`;
  LOGS.html = "";
  $$("#lg-src [data-s]").forEach((b) => b.addEventListener("click", () => { LOGS.src = b.dataset.s; $$("#lg-src button").forEach((x) => x.classList.toggle("on", x === b)); LOGS.html = ""; logsData(true); }));
  let qt = 0;
  $("#lg-q").addEventListener("input", (e) => { clearTimeout(qt); qt = setTimeout(() => { LOGS.q = e.target.value; LOGS.html = ""; logsData(true); }, 250); });
  $("#lg-auto").addEventListener("change", (e) => { LOGS.auto = e.target.checked; });
  $("#lg-copy").addEventListener("click", () => copyText($("#logbox").innerText).then(() => toast("Лог скопирован"), () => toast("Не получилось скопировать", true)));
  logsData(true);
}

const logLine = (l) => {
  const m = /^(\d{4}\/\d\d\/\d\d |\[[\d:]+\] \[)?(\S*\s?\d\d:\d\d:\d\d(?:\.\d+)?\]?)?(.*)$/.exec(l) || [];
  const bad = /ошибк|error|fail|не получилось|panic|не удалось|не открылся/i.test(l), warn = /warn|предупрежд|повтор|ложное/i.test(l);
  return `<div class="ll${bad ? " bad" : warn ? " warn" : ""}"><span class="lt">${esc((m[1] || "") + (m[2] || ""))}</span>${esc(m[3] ?? l)}</div>`;
};

// логи на главной: карточка занимает остаток правой колонки; места меньше ~170 px — её не видно и лог не грузим
const HL = { at: 0, html: "", on: false };
function homeLogsSetup() {
  const c = $("#hlogs");
  if (!c) return;
  HL.html = ""; HL.on = false;
  new ResizeObserver(() => {
    const on = c.isConnected && getComputedStyle(c).display !== "none" && c.clientHeight >= 170;
    c.classList.toggle("tight", !on);
    if (on && !HL.on) { HL.on = true; homeLogs(); } else HL.on = on;
  }).observe(c);
}
async function homeLogs() {
  HL.at = Date.now();
  const box = $("#hlogbox");
  if (!box || !HL.on) return;
  const n = Math.max(10, Math.ceil(box.clientHeight / 19) + 4);
  let d;
  try { d = await api("GET", `/api/logs?src=jarvis&n=${n}`); } catch { return; }
  const html = (d.lines || []).map(logLineShort).join("") || `<p class="empty">${esc(d.error || "Пока пусто")}</p>`;
  if (html === HL.html || !box.isConnected) return;
  HL.html = html;
  box.innerHTML = html;
  box.scrollTop = box.scrollHeight;
}

// короткая строка для главной: время — только минуты и секунды
const logLineShort = (l) => {
  const m = /^\d{4}\/\d\d\/\d\d \d\d:(\d\d:\d\d)(?:\.\d+)? (.*)$/.exec(l);
  if (!m) return logLine(l);
  const bad = /ошибк|error|fail|не получилось|panic|не удалось|не открылся/i.test(m[2]), warn = /warn|предупрежд|повтор|ложное/i.test(m[2]);
  return `<div class="ll${bad ? " bad" : warn ? " warn" : ""}"><span class="lt">${m[1]}</span>${esc(m[2])}</div>`;
};

async function logsData(force) {
  if (page !== "logs") return;
  LOGS.at = Date.now();
  let d;
  try { d = await api("GET", `/api/logs?src=${LOGS.src}&n=400&q=${encodeURIComponent(LOGS.q)}`); } catch { return; }
  const box = $("#logbox");
  if (!box) return;
  const html = (d.lines || []).map(logLine).join("") || `<p class="empty">${esc(d.error || (LOGS.q ? "Ничего не нашлось" : "Пока пусто"))}</p>`;
  if (html === LOGS.html && !force) return;
  const pinned = box.scrollHeight - box.scrollTop - box.clientHeight < 60 || force;
  LOGS.html = html;
  box.innerHTML = html;
  if (pinned) box.scrollTop = box.scrollHeight;
}

// ---------------------------------------------------------------- характеры: карточки + редактор поверх страницы
const CHAR_PRESETS = [
  ["Дворецкий", "Ты — Джарвис, голосовой помощник в умной колонке. Говоришь по-русски, коротко и по делу, с лёгкой иронией, как дворецкий из фильма. Обращаешься к хозяину на «ты»."],
  ["Кореш", "Ты — Джарвис, но общаешься как близкий друг: на «ты», с юмором и подколами, живо и по-простому. Говоришь коротко и честно: не знаешь — так и скажи."],
  ["Учёный", "Ты — Джарвис, спокойный и точный, как учёный. Объясняешь просто, но без ошибок, приводишь одну яркую деталь или цифру. Отвечаешь в 2–3 предложения, если не просят подробнее."],
  ["Лаконичный", "Ты — Джарвис. Отвечай максимально коротко: одно предложение, без вступлений и вежливых оборотов. Только суть."],
  ["Детский", "Ты — Джарвис, добрый помощник для ребёнка. Говоришь просто, тепло и весело, без сложных слов и без страшного. Хвалишь за интересные вопросы."],
];

function viewCharacters() {
  $("#view").innerHTML = `<header class="ph ph-row"><div><h1>Характеры</h1><p class="sub">Кто такой Джарвис и как он с вами общается. Характер выбирается в настройках ассистента</p></div>
    <div class="actions"><button class="btn primary" id="ch-add">${icon("plus")}Новый характер</button></div></header>
  <div class="chars stagger" id="chars"></div>
  <div class="modal hidden" id="ch-modal" role="dialog" aria-modal="true" aria-labelledby="ch-mt"><div class="modal-bg" data-close></div>
    <div class="modal-card">
      <div class="card-head"><h2 id="ch-mt">Новый характер</h2><button class="ib" type="button" data-close aria-label="Закрыть">${icon("x")}</button></div>
      <div class="f"><label>Название</label><input type="text" id="c-name" maxlength="40" placeholder="Например, Дворецкий" autocomplete="off"></div>
      <div class="f"><label>Начать с образца</label><div class="presets-row" id="ch-pre">${CHAR_PRESETS.map(([n], i) => `<button type="button" class="chip" data-p="${i}">${esc(n)}</button>`).join("")}</div></div>
      <div class="f"><label>Как он говорит <span class="meta" id="c-count"></span></label><textarea class="big" id="c-text" placeholder="Ты — Джарвис… Опишите манеру: тон, обращение, длину ответов, любимые словечки"></textarea></div>
      <p class="note">Пишите от второго лица: «Ты — …». Не вставляйте ключи и пароли — текст уходит модели с каждым разговором.</p>
      <div class="row"><button class="btn primary" id="c-save">Сохранить</button><button class="btn ghost" type="button" data-close>Отмена</button><span class="spacer"></span><button class="btn ghost danger hidden" id="c-del">Удалить</button></div>
    </div></div>`;
  let editing = null;
  const modal = $("#ch-modal");
  const count = () => { $("#c-count").textContent = `${$("#c-text").value.length} симв.`; };
  const open = (c) => {
    editing = c || null;
    $("#ch-mt").textContent = c ? `Изменить «${c.name}»` : "Новый характер";
    $("#c-name").value = c?.name || ""; $("#c-text").value = c?.text || "";
    const used = c && S.profiles.some((p) => p.prompt === c.id);
    $("#c-del").classList.toggle("hidden", !c || used);
    count();
    modal.classList.remove("hidden", "out"); void modal.offsetWidth; modal.classList.add("in");
    setTimeout(() => $("#c-name").focus(), 60);
  };
  const close = () => { modal.classList.remove("in"); modal.classList.add("out"); dirty = false; setTimeout(() => { if (modal.classList.contains("out")) modal.classList.add("hidden"); }, 260); };
  const draw = () => {
    $("#chars").innerHTML = S.prompts.map((c) => {
      const used = S.profiles.filter((p) => p.prompt === c.id);
      return `<article class="card char" data-id="${esc(c.id)}" tabindex="0">
        <div class="char-h"><span class="char-av">${esc((c.name || "?").trim()[0] || "?")}</span><div class="char-n"><h2>${esc(c.name)}</h2>
          <span class="meta">${used.length ? `у ${used.map((p) => esc(p.name)).join(", ")}` : "пока ни у кого"}</span></div></div>
        <p class="char-t">${esc(c.text)}</p>
        <div class="char-f"><span class="meta">${c.text.length} симв.</span><span class="spacer"></span><span class="char-edit">${icon("tune")}Изменить</span></div>
      </article>`;
    }).join("") + `<button class="char char-new" type="button" id="ch-new">${icon("plus")}<b>Новый характер</b><small>Опишите манеру — и Джарвис заговорит по-другому</small></button>`;
  };
  draw();
  $("#chars").addEventListener("click", (e) => {
    if (e.target.closest("#ch-new")) return open();
    const card = e.target.closest(".char[data-id]");
    if (card) open(S.prompts.find((c) => c.id === card.dataset.id));
  });
  $("#chars").addEventListener("keydown", (e) => { const card = e.target.closest(".char[data-id]"); if (card && (e.key === "Enter" || e.key === " ")) { e.preventDefault(); open(S.prompts.find((c) => c.id === card.dataset.id)); } });
  $("#ch-add").addEventListener("click", () => open());
  modal.addEventListener("click", (e) => { if (e.target.closest("[data-close]")) close(); });
  modal.addEventListener("keydown", (e) => { if (e.key === "Escape") close(); });
  $("#ch-pre").addEventListener("click", (e) => {
    const b = e.target.closest("[data-p]"); if (!b) return;
    const [n, t] = CHAR_PRESETS[+b.dataset.p];
    if ($("#c-text").value.trim() && !confirm("Заменить текст образцом?")) return;
    $("#c-text").value = t; if (!$("#c-name").value.trim()) $("#c-name").value = n;
    dirty = true; count();
  });
  $("#c-text").addEventListener("input", () => { dirty = true; count(); });
  $("#c-name").addEventListener("input", () => { dirty = true; });
  $("#c-save").addEventListener("click", (e) => act(e.target, async () => {
    await api("PUT", "/api/prompt", { id: editing?.id || "", name: $("#c-name").value.trim(), text: $("#c-text").value.trim() });
    dirty = false; await reload(); draw(); close();
  }, "Характер сохранён"));
  $("#c-del").addEventListener("click", (e) => { if (editing && confirm(`Удалить характер «${editing.name}»?`)) act(e.target, async () => { await api("DELETE", "/api/prompt?id=" + encodeURIComponent(editing.id)); await reload(); draw(); close(); }, "Удалено"); });
}

// ---------------------------------------------------------------- обновления (на странице «Безопасность»)
async function drawUpdate() {
  const box = $("#upd");
  if (!box) return;
  let U;
  try { U = await api("GET", "/api/update"); } catch (e) { box.innerHTML = `<p class="err">${esc(e.message)}</p>`; return; }
  const auto = U.check || U.auto;
  box.innerHTML = `<div class="row upd-row">
      <span>Версия <b>${esc(U.version)}</b></span>
      <a class="btn sm ghost" href="https://github.com/newfpv/YaJavis/releases" target="_blank" rel="noopener noreferrer">GitHub Releases</a>
      ${U.busy ? `<span class="note"><span class="spin"></span>${esc(U.busy)}…</span>`
        : U.available ? `<span class="st ok">вышла ${esc(U.latest)}</span><button class="btn sm primary" id="u-install">Обновить</button>`
        : U.latest ? `<span class="st ok">последняя</span>` : ""}
      <span class="spacer"></span>
      ${U.repo ? `<button class="btn sm" id="u-checknow">Проверить</button>` : ""}
      <label class="btn sm ghost" for="u-file" data-tip="Поставить файл ${esc(U.asset)} с компьютера">Файлом…</label><input type="file" id="u-file" class="hidden">
    </div>
    <div class="seg" id="u-mode"><button type="button" data-auto="0" class="${auto ? "" : "on"}">Вручную</button><button type="button" data-auto="1" class="${auto ? "on" : ""}">Автообновление</button></div>
    <small class="muted" id="u-hint"></small>
    <label class="sw hu u-stats"><input type="checkbox" id="u-stats"${U.stats ? " checked" : ""}><span class="tr"></span>
      <span class="hu-t"><b>Счётчик установок</b><small>Раз на версию колонка скачивает пустой файл с GitHub — так в README видно, сколько колонок с Джарвисом. Передаются только модель и версия.</small></span></label>
    ${U.error ? `<p class="err">${esc(U.error)}</p>` : ""}
    ${U.available && U.notes ? `<details class="mini"><summary>Что нового в ${esc(U.latest)}</summary><pre class="unotes">${esc(U.notes)}</pre></details>` : ""}`;
  const hint = (on) => { $("#u-hint").textContent = !U.repo ? "Укажите репозиторий обновлений или выберите файл с компьютера." : on ? `Новая версия ставится сама ночью, с 3 до 6, когда колонка молчит${U.checked ? ` · проверено ${U.checked}` : ""}` : "Только когда нажмёте «Проверить» или выберете файл"; };
  hint(auto);
  $$("#u-mode [data-auto]").forEach((b) => b.addEventListener("click", () => {
    const on = b.dataset.auto === "1";
    $$("#u-mode button").forEach((x) => x.classList.toggle("on", x === b));
    hint(on);
    api("PUT", "/api/update", { check: on, auto: on }).then(() => toast(on ? "Автообновление включено" : "Обновления — вручную")).catch((err) => toast(err.message, true));
  }));
  $("#u-stats").addEventListener("change", (e) => api("PUT", "/api/update", { stats: e.target.checked }).then(() => toast(e.target.checked ? "Счётчик включён" : "Счётчик выключен")).catch((err) => toast(err.message, true)));
  $("#u-checknow")?.addEventListener("click", (e) => act(e.target, async () => { await api("POST", "/api/update/check"); drawUpdate(); }));
  $("#u-install")?.addEventListener("click", (e) => act(e.target, async () => { e.target.textContent = "Скачиваю…"; await api("POST", "/api/update/install"); await waitRestart("Обновление скачано — Джарвис перезапускается"); }));
  $("#u-file").addEventListener("change", (e) => { const file = e.target.files[0]; if (!file) return; act(null, async () => { toast("Загружаю на колонку…"); await upload("PUT", "/api/update/upload", file); await waitRestart("Файл принят — Джарвис перезапускается"); }); });
}

// ---------------------------------------------------------------- фон комнаты на Google Диск — в два клика
let BG = null, bgServer = false, bgHours = 4;
const hm = (min) => min >= 60 ? `${Math.floor(min / 60)} ч ${min % 60} мин` : `${min} мин`;

async function drawBg() {
  const box = $("#bg");
  if (!box) return;
  try { BG = await api("GET", "/api/bgrec"); } catch (e) { box.innerHTML = `<p class="err">${esc(e.message)}</p>`; return; }
  const g = BG.gdrive || {}, server = bgServer || BG.to === "server";
  const ready = server ? !!BG.url : g.signed_in;
  let target;
  if (server) {
    target = `<div class="grid g2"><input type="text" id="bg-url" value="${esc(BG.url)}" placeholder="https://webdav.yandex.ru/jarvis-bg или http://192.168.1.10:8765/"><div class="row nw"><input type="text" id="bg-user" value="${esc(BG.user)}" placeholder="Логин" autocomplete="off"><input type="password" id="bg-pass" placeholder="${BG.pass_set ? "пароль сохранён" : "Пароль"}" autocomplete="new-password"></div></div>
      <div class="row"><button class="btn sm" id="bg-save">Сохранить</button><small class="muted">Яндекс Диск — адрес webdav.yandex.ru/папка и пароль приложения; компьютер — <code>python kws/bg_receiver.py</code></small><span class="spacer"></span><button class="btn sm ghost" id="bg-to-drive">Лучше Google Диск</button></div>`;
  } else if (g.signed_in) {
    target = `<div class="row"><span class="st ok">Google Диск подключён — папка «Jarvis — фон комнаты»</span><span class="spacer"></span><button class="btn sm ghost danger" id="gd-out">Отключить</button></div>`;
  } else if (g.code) {
    target = `<div class="gcode"><span class="gstep">1</span><a class="btn primary" href="${esc(g.url)}" target="_blank" rel="noopener">Открыть ${esc(g.url.replace("https://www.", "").replace("https://", ""))}</a>
      <span class="gstep">2</span><span>Введите код <b class="code-big">${esc(g.code)}</b> и разрешите доступ</span></div>
      <small class="muted">Колонка увидит только свою папку на Диске. Эта страница сама поймёт, что вы вошли.</small>`;
  } else if (!g.client_set) {
    target = `<p class="note">Google Диск не настроен в этой сборке. Можно использовать свой сервер или установить официальный релиз YaJavis с подключением Google Диска.</p>
      <div class="row"><button class="btn sm ghost" id="bg-to-server">Отправлять на свой сервер</button></div>`;
  } else {
    target = `${g.error ? `<p class="err">${esc(g.error)}</p>` : ""}<div class="row"><button class="btn primary" id="gd-in">Подключить Google Диск</button><small class="muted">откроется страница Google — войдёте и разрешите доступ</small><span class="spacer"></span><button class="btn sm ghost" id="bg-to-server">или свой сервер</button></div>`;
  }
  box.innerHTML = `${target}
    <div class="bg-run${ready ? "" : " off"}">${BG.running
      ? `<span class="rec-dot"></span><span><b>Пишу фон</b> · осталось ${hm(BG.left_min)} · отправлено ${BG.sent} мин (${BG.sent_mb.toFixed(0)} МБ)${BG.dropped ? ` · пропущено ${BG.dropped}` : ""}</span><span class="spacer"></span><button class="btn sm" id="bg-stop">Остановить</button>`
      : `<div class="seg" id="bg-hours">${[1, 2, 4, 8, 12].map((h) => `<button type="button" data-h="${h}" class="${h === bgHours ? "on" : ""}">${h} ч</button>`).join("")}</div><span class="spacer"></span><button class="btn primary" id="bg-start"${ready ? "" : " disabled"}>Записать ${bgHours} ч фона</button>`}</div>
    ${BG.error ? `<p class="err">${esc(BG.error)}</p>` : ""}`;
  const to = (t) => api("PUT", "/api/bgrec", { to: t }).then(drawBg).catch((e) => toast(e.message, true));
  $("#bg-to-server")?.addEventListener("click", () => { bgServer = true; to("server"); });
  $("#bg-to-drive")?.addEventListener("click", () => { bgServer = false; to("gdrive"); });
  $$("#bg-hours [data-h]").forEach((b) => b.addEventListener("click", () => { bgHours = +b.dataset.h; $$("#bg-hours button").forEach((x) => x.classList.toggle("on", x === b)); $("#bg-start").textContent = `Записать ${bgHours} ч фона`; }));
  $("#gd-in")?.addEventListener("click", (e) => act(e.target, async () => { await api("POST", "/api/gdrive/login"); drawBg(); }));
  $("#gd-out")?.addEventListener("click", (e) => { if (confirm("Отключить Google Диск?")) act(e.target, async () => { await api("DELETE", "/api/gdrive/login"); drawBg(); }); });
  $("#bg-save")?.addEventListener("click", (e) => act(e.target, async () => { const body = { to: "server", url: $("#bg-url").value, user: $("#bg-user").value }; if ($("#bg-pass").value) body.pass = $("#bg-pass").value; await api("PUT", "/api/bgrec", body); drawBg(); }, "Сохранено"));
  $("#bg-start")?.addEventListener("click", (e) => act(e.target, async () => { await api("POST", "/api/bgrec/start", { hours: bgHours }); drawBg(); }, "Запись фона началась"));
  $("#bg-stop")?.addEventListener("click", (e) => act(e.target, async () => { await api("POST", "/api/bgrec/stop"); drawBg(); }));
  clearTimeout(drawBg.t);
  if (BG.running || g.code) drawBg.t = setTimeout(() => { if (page === "training") drawBg(); }, g.code ? 3000 : 15000);
}

// ---------------------------------------------------------------- пружины (как в designbookmark motion: stiffness/damping)
const Motion = (() => {
  const reduce = matchMedia("(prefers-reduced-motion: reduce)").matches;
  const S = (x, k = 400, c = 30) => ({ x, v: 0, t: x, k, c: reduce ? 2 * Math.sqrt(k) : c });
  const step = (s, dt) => { const a = -s.k * (s.x - s.t) - s.c * s.v; s.v += a * dt; s.x += s.v * dt; };
  const settled = (s) => Math.abs(s.x - s.t) < 0.002 && Math.abs(s.v) < 0.01;
  const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
  const timers = [], comps = new Set();
  const after = (sec, fn) => { timers.push({ t: sec, fn }); };
  let last = performance.now(), running = false;
  function loop(now) {
    const dt = Math.min((now - last) / 1000, 1 / 30); last = now;
    for (let i = timers.length - 1; i >= 0; i--) if ((timers[i].t -= dt) <= 0) timers.splice(i, 1)[0].fn();
    for (let k = 0; k < 4; k++) comps.forEach((c) => c.frame(dt / 4));
    comps.forEach((c) => c.render());
    comps.forEach((c) => { if (c.done && c.done()) comps.delete(c); });   // отыгравшее — из цикла
    if (comps.size || timers.length) requestAnimationFrame(loop); else running = false;
  }
  const wake = () => { if (!running) { running = true; last = performance.now(); requestAnimationFrame(loop); } };
  return { S, step, settled, clamp, reduce, after: (s, f) => { after(s, f); wake(); }, add: (c) => { comps.add(c); wake(); }, remove: (c) => comps.delete(c) };
})();

// ---------------------------------------------------------------- ответ Джарвиса «печатается»: слова всплывают по одному
// Пузырь открывается из пилюли «Думаю» и растёт пружиной (260/26); каждое слово поднимается и проясняется
// на своей пружине (340/26) с шагом 70 мс — как Streaming reply из designbookmark.
const shownWords = {};   // ts реплики → сколько слов ответа уже показано
const bubbleSize = {};   // ts → последний размер пузыря (из него растёт следующий)

function answerHtml(ts, text, animate) {
  const words = text.split(/\s+/).filter(Boolean);
  const from = animate ? Math.min(shownWords[ts] ?? 0, words.length) : words.length;
  shownWords[ts] = words.length;
  if (from >= words.length) return esc(text);
  return esc(words.slice(0, from).join(" ")) + (from ? " " : "") + words.slice(from).map((w) => `<span class="wd">${esc(w)}</span>`).join(" ");
}

function streamBubbles(feed) {
  for (const b of feed.querySelectorAll(".b.him[data-k]")) {
    const k = b.dataset.k, natural = { w: b.offsetWidth, h: b.offsetHeight };
    const prev = bubbleSize[k];
    bubbleSize[k] = natural;
    const words = [...b.querySelectorAll(".wd")];
    if (Motion.reduce || (!words.length && (!prev || (prev.w === natural.w && prev.h === natural.h)))) continue;
    // размер: из прежнего (пилюля «Думаю» или прошлый кусок ответа) в новый
    const from = prev || { w: Math.min(natural.w, 96), h: Math.min(natural.h, 36) };
    const W = Motion.S(from.w, 260, 26), H = Motion.S(from.h, 260, 26);
    W.t = natural.w; H.t = words.length ? Math.max(from.h, Math.min(natural.h, 36)) : natural.h;
    b.classList.add("growing");
    const ws = words.map((el) => ({ el, s: Motion.S(0, 340, 26), on: false }));
    // высота растёт строка за строкой — вслед за последним появившимся словом
    ws.forEach((w, i) => { w.el.style.opacity = 0; Motion.after(i * 0.07, () => {
      w.on = true; w.s.t = 1;
      H.t = i === ws.length - 1 ? natural.h : Math.min(natural.h, Math.max(H.t, w.el.offsetTop + w.el.offsetHeight + 8));
    }); });
    const c = {
      frame(dt) { Motion.step(W, dt); Motion.step(H, dt); ws.forEach((w) => w.on && Motion.step(w.s, dt)); },
      render() {
        if (!b.isConnected) return;
        b.style.width = W.x.toFixed(1) + "px"; b.style.height = H.x.toFixed(1) + "px";
        for (const w of ws) {
          if (!w.on) continue;
          const v = Motion.clamp(w.s.x, 0, 1.2), o = Motion.clamp(v, 0, 1);
          w.el.style.opacity = o.toFixed(3);
          w.el.style.transform = `translateY(${((1 - w.s.x) * 6).toFixed(2)}px)`;
          w.el.style.filter = v < 0.98 ? `blur(${((1 - o) * 4).toFixed(2)}px)` : "none";
        }
      },
      done() {
        const fin = !b.isConnected || (Motion.settled(W) && Motion.settled(H) && ws.every((w) => w.on && Motion.settled(w.s)));
        if (fin && b.isConnected) { b.style.width = b.style.height = ""; b.classList.remove("growing"); ws.forEach((w) => { w.el.style.cssText = ""; }); }
        return fin;
      },
    };
    Motion.add(c);
  }
}

// ---------------------------------------------------------------- «желейный» ползунок (designbookmark Jelly slider)
// Кружок идёт за пальцем на пружине (900/50) и растягивается от скорости; при перетаскивании из него
// поднимается капля со значением (420/22) — через goo-фильтр они сливаются, как жидкость. За краем —
// резиновое сопротивление и отскок (300/14). Хвататься можно за любое место полосы.
function jelly(root, o) {
  const R = 9, TR = 11, D = 16, OX = 32, OY = 64, CY = 16, RISE = -34, REST = 0.45;
  root.classList.add("jelly");
  root.innerHTML = `<div class="j-track"><div class="j-fill"></div></div>
    <div class="j-goo"><div class="j-blob j-drop"></div><div class="j-blob j-thumb"></div></div><div class="j-label"></div>`;
  root.tabIndex = 0;
  root.setAttribute("role", "slider");
  root.setAttribute("aria-label", o.label || "");
  const fill = $(".j-fill", root), thumb = $(".j-thumb", root), drop = $(".j-drop", root), lab = $(".j-label", root);
  let W = root.clientWidth || 0, min = o.min ?? 0, max = o.max ?? 100;
  const span = () => Math.max(W - 2 * R, 1);
  const toX = (v) => R + ((v - min) / Math.max(max - min, 1e-9)) * span();
  const toV = (x) => min + Motion.clamp((x - R) / span(), 0, 1) * (max - min);
  const snap = (v) => o.step ? Math.round(v / o.step) * o.step : v;
  const X = Motion.S(toX(o.value ?? min), 900, 50), DX = Motion.S(X.x, 260, 18), DY = Motion.S(CY, 420, 22), DS = Motion.S(REST, 420, 22), PS = Motion.S(1, 520, 16);
  const tune = (s, k, c) => { s.k = k; s.c = Motion.reduce ? 2 * Math.sqrt(k) : c; };
  const rubber = (x, a, b, d = 18) => x < a ? a - (1 - 1 / ((a - x) * 0.55 / d + 1)) * d : x > b ? b + (1 - 1 / ((x - b) * 0.55 / d + 1)) * d : x;
  const localX = (e) => e.clientX - root.getBoundingClientRect().left;
  let dragging = false, hover = false, lifted = false, lastSent = null;
  const sync = () => { const up = dragging || lifted; DY.t = up ? CY + RISE : CY; DS.t = up ? 1 : REST; PS.t = dragging ? 1.18 : hover ? 1.1 : 1; Motion.add(anim); };
  const value = () => snap(toV(Motion.clamp(X.t, R, W - R)));

  root.addEventListener("pointerdown", (e) => {
    if (o.disabled?.()) return;
    dragging = true; try { root.setPointerCapture(e.pointerId); } catch { /* */ }
    tune(X, 900, 50); X.t = rubber(localX(e), R, W - R); sync();
  });
  root.addEventListener("pointermove", (e) => {
    if (!dragging) return;
    X.t = rubber(localX(e), R, W - R);
    const v = value(); if (v !== lastSent) { lastSent = v; o.onInput?.(v); }
  });
  const end = () => {
    if (!dragging) return;
    dragging = false; tune(X, 300, 14); X.t = Motion.clamp(X.t, R, W - R);
    if (o.step) X.t = toX(value());
    sync(); o.onChange?.(value());
  };
  root.addEventListener("pointerup", end); root.addEventListener("pointercancel", end);
  root.addEventListener("pointerenter", (e) => { if (e.pointerType === "mouse") { hover = true; sync(); } });
  root.addEventListener("pointerleave", () => { hover = false; sync(); });
  root.addEventListener("keydown", (e) => {
    const k = o.keyStep || o.step || (max - min) / 20;
    const d = { ArrowRight: k, ArrowUp: k, ArrowLeft: -k, ArrowDown: -k }[e.key];
    if (d === undefined || o.disabled?.()) return;
    e.preventDefault();
    const v = snap(Motion.clamp(value() + d, min, max));
    tune(X, 420, 34); X.t = toX(v); lifted = true; sync();
    o.onInput?.(v); clearTimeout(root.kt); root.kt = setTimeout(() => o.onChange?.(v), 350);
  });
  root.addEventListener("blur", () => { lifted = false; sync(); });
  new ResizeObserver(() => { const w = root.clientWidth; if (!w || w === W) return; const v = toV(X.t); W = w; X.x = X.t = DX.x = DX.t = toX(v); X.v = DX.v = 0; Motion.add(anim); }).observe(root);

  const anim = {
    frame(dt) { DX.t = X.x; Motion.step(X, dt); Motion.step(DX, dt); Motion.step(DY, dt); Motion.step(DS, dt); Motion.step(PS, dt); },
    render() {
      const x = X.x, st = 1 + Math.min(Math.abs(X.v) / 2400, 0.4);
      thumb.style.transform = `translate(${(x - TR + OX).toFixed(2)}px,${(CY - TR + OY).toFixed(2)}px) scale(${(st * PS.x).toFixed(3)},${(PS.x / st).toFixed(3)})`;
      const tilt = Motion.clamp((DX.x - X.x) * 0.9, -28, 28);
      const tr = `translate(${(DX.x - D).toFixed(2)}px,${(DY.x - D).toFixed(2)}px) rotate(${tilt.toFixed(2)}deg) scale(${DS.x.toFixed(3)})`;
      drop.style.transform = `translate(${OX}px,${OY}px) ` + tr;
      lab.style.transform = tr;
      lab.style.opacity = Motion.clamp((DS.x - 0.7) / 0.25, 0, 1).toFixed(3);
      lab.textContent = (o.format || String)(snap(toV(x)));
      fill.style.width = Math.max(0, x).toFixed(2) + "px";
      root.setAttribute("aria-valuenow", String(snap(toV(x))));
    },
    done() { return !root.isConnected || (!dragging && !lifted && [X, DX, DY, DS, PS].every(Motion.settled)); },
  };
  Motion.add(anim);
  return {
    // значение снаружи (например, ход трека): пока человек тянет — не мешаем
    set(v, glide = false) {
      if (dragging) return;
      v = Motion.clamp(v, min, max);
      if (glide) { tune(X, 420, 34); X.t = toX(v); } else { X.x = X.t = toX(v); X.v = 0; }
      Motion.add(anim);
    },
    range(a, b) { min = a; max = Math.max(b, a + 1e-9); },
    get dragging() { return dragging; },
  };
}

// ---------------------------------------------------------------- плавная плашка у переключателей (Tab pill)
function pills() {
  for (const seg of document.querySelectorAll("#view .seg, .gate .seg")) {
    let pill = seg.querySelector(":scope > .seg-pill");
    if (!pill) { pill = document.createElement("span"); pill.className = "seg-pill"; seg.prepend(pill); seg.classList.add("has-pill"); seg.dataset.fresh = "1"; }
    const on = seg.querySelector(":scope > button.on");
    if (!on) { pill.style.opacity = "0"; continue; }
    if (seg.dataset.fresh) { pill.classList.add("instant"); delete seg.dataset.fresh; } else pill.classList.remove("instant");
    pill.style.opacity = "1";
    pill.style.width = on.offsetWidth + "px"; pill.style.height = on.offsetHeight + "px";
    pill.style.transform = `translate(${on.offsetLeft}px, ${on.offsetTop}px)`;
  }
}
let pillQueued = false;
new MutationObserver(() => { if (!pillQueued) { pillQueued = true; requestAnimationFrame(() => { pillQueued = false; pills(); }); } })
  .observe(document.getElementById("view"), { subtree: true, childList: true, attributes: true, attributeFilter: ["class"] });
addEventListener("resize", () => pills());

// ---------------------------------------------------------------- запуск
start().catch(() => {});
