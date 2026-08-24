import './style.css'
import { steps, actorInfo, clientHolding } from './flow.js'

const app = document.querySelector('#app')

app.innerHTML = `
<header>
  <h1>OAuth 2.0 — Client Credentials Grant</h1>
  <p class="subtitle">RFC 6749 §4.4 — machine-to-machine, no user, no redirect.</p>
</header>

<div class="stage">
  <svg class="arrows"></svg>
  <div class="actor" data-actor="client" tabindex="0">
    <div class="actor-icon">🖥️</div>
    <div class="actor-label">Client</div>
  </div>
  <div class="actor" data-actor="auth" tabindex="0">
    <div class="actor-icon">🔐</div>
    <div class="actor-label">Auth Server</div>
  </div>
  <div class="actor" data-actor="resource" tabindex="0">
    <div class="actor-icon">📦</div>
    <div class="actor-label">Resource Server</div>
  </div>
</div>

<div class="info-panel" id="info-panel" hidden>
  <button class="info-close" id="info-close">✕</button>
  <h3 id="info-title"></h3>
  <div id="info-body"></div>
</div>

<div class="panel">
  <div class="panel-head">
    <span class="step-count" id="step-count"></span>
    <h2 id="step-title"></h2>
  </div>
  <p id="step-detail"></p>
  <pre id="step-http"></pre>
</div>

<div class="controls">
  <button id="reset">↺ Reset</button>
  <button id="prev">◀ Prev</button>
  <button id="play">▶ Play</button>
  <button id="next">Next ▶</button>
</div>
`

const els = {
  count: document.querySelector('#step-count'),
  title: document.querySelector('#step-title'),
  detail: document.querySelector('#step-detail'),
  http: document.querySelector('#step-http'),
  prev: document.querySelector('#prev'),
  next: document.querySelector('#next'),
  play: document.querySelector('#play'),
  reset: document.querySelector('#reset'),
  actors: document.querySelectorAll('.actor'),
  infoPanel: document.querySelector('#info-panel'),
  infoTitle: document.querySelector('#info-title'),
  infoBody: document.querySelector('#info-body'),
  infoClose: document.querySelector('#info-close'),
}

const stage = document.querySelector('.stage')
const svg = document.querySelector('.arrows')
const actorEls = { client: els.actors[0], auth: els.actors[1], resource: els.actors[2] }

function center(actor) {
  const stageBox = stage.getBoundingClientRect()
  const box = actorEls[actor].getBoundingClientRect()
  return {
    x: box.left + box.width / 2 - stageBox.left,
    y: box.top + box.height / 2 - stageBox.top,
  }
}

function drawEdges() {
  svg.innerHTML = ''
  const c = center('client')
  const a = center('auth')
  const r = center('resource')
  for (const [p1, p2] of [[c, a], [c, r]]) {
    const line = document.createElementNS('http://www.w3.org/2000/svg', 'line')
    line.setAttribute('x1', p1.x)
    line.setAttribute('y1', p1.y)
    line.setAttribute('x2', p2.x)
    line.setAttribute('y2', p2.y)
    line.setAttribute('class', 'arrow-line')
    svg.appendChild(line)
  }
}

let i = 0
let timer = null
let openActor = null

function renderInfo() {
  if (!openActor) {
    els.infoPanel.hidden = true
    return
  }
  els.infoPanel.hidden = false
  if (openActor === 'client') {
    els.infoTitle.textContent = 'Client — holding'
    els.infoBody.innerHTML = `<ul>${clientHolding(i).map((x) => `<li>${x}</li>`).join('')}</ul>`
  } else {
    const info = actorInfo[openActor]
    els.infoTitle.textContent = `${info.label} — ${info.host}`
    els.infoBody.innerHTML = `
      <p class="info-section-label">Endpoints</p>
      <ul>${info.endpoints.map((x) => `<li>${x}</li>`).join('')}</ul>
      <p class="info-section-label">Holds</p>
      <ul>${info.holds.map((x) => `<li>${x}</li>`).join('')}</ul>
    `
  }
}

function render() {
  const s = steps[i]
  els.count.textContent = `Step ${i + 1} / ${steps.length}`
  els.title.textContent = s.title
  els.detail.textContent = s.detail
  els.http.textContent = s.http

  els.actors.forEach((el) => {
    el.classList.toggle('active', el.dataset.actor === s.from || el.dataset.actor === s.to)
  })

  svg.querySelectorAll('.pulse').forEach((n) => n.remove())
  if (s.from !== s.to) {
    const p1 = center(s.from)
    const p2 = center(s.to)
    const dot = document.createElementNS('http://www.w3.org/2000/svg', 'circle')
    dot.setAttribute('r', 6)
    dot.setAttribute('cx', p1.x)
    dot.setAttribute('cy', p1.y)
    dot.setAttribute('class', 'pulse')
    svg.appendChild(dot)
    dot.animate(
      [
        { cx: p1.x, cy: p1.y, opacity: 0 },
        { cx: p1.x, cy: p1.y, opacity: 1, offset: 0.15 },
        { cx: p2.x, cy: p2.y, opacity: 1 },
      ],
      { duration: 900, easing: 'ease-in-out', fill: 'forwards' },
    )
  }

  els.prev.disabled = i === 0
  els.next.disabled = i === steps.length - 1

  renderInfo()
}

els.actors.forEach((el) => {
  el.addEventListener('click', () => {
    const actor = el.dataset.actor
    openActor = openActor === actor ? null : actor
    renderInfo()
  })
})
els.infoClose.addEventListener('click', () => {
  openActor = null
  renderInfo()
})

function stopPlay() {
  if (timer) {
    clearInterval(timer)
    timer = null
    els.play.textContent = '▶ Play'
  }
}

els.reset.addEventListener('click', () => {
  stopPlay()
  i = 0
  render()
})
els.prev.addEventListener('click', () => {
  i = Math.max(0, i - 1)
  render()
})
els.next.addEventListener('click', () => {
  i = Math.min(steps.length - 1, i + 1)
  render()
})
els.play.addEventListener('click', () => {
  if (timer) {
    stopPlay()
    return
  }
  els.play.textContent = '⏸ Pause'
  timer = setInterval(() => {
    if (i >= steps.length - 1) {
      stopPlay()
      return
    }
    i += 1
    render()
  }, 1800)
})

function syncSvgSize() {
  const box = stage.getBoundingClientRect()
  svg.setAttribute('viewBox', `0 0 ${box.width} ${box.height}`)
  drawEdges()
}

window.addEventListener('resize', () => {
  syncSvgSize()
  render()
})

syncSvgSize()
render()
