// SPDX-License-Identifier: Apache-2.0
// Berichte: draws each [data-chart] from the JSON the server put into the page
// (#report-data, by card id) with d3. The same numbers stay readable in the
// table below each chart, so the charts are aria-hidden.
(() => {
  const d3 = window.d3;
  const source = document.getElementById('report-data');
  if (!d3 || !source) return;
  const data = JSON.parse(source.textContent || '{}');
  const root = getComputedStyle(document.documentElement);
  const css = (name) => root.getPropertyValue(name).trim();
  const colors = [1, 2, 3, 4, 5, 6].map((i) => css(`--chart-${i}`));
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  const tip = document.createElement('div');
  tip.className = 'report-tip';
  tip.setAttribute('role', 'presentation');
  document.body.append(tip);
  const show = (e, html) => {
    tip.innerHTML = html;
    tip.style.left = `${e.clientX + 12}px`;
    tip.style.top = `${e.clientY + 12}px`;
    tip.style.opacity = 1;
  };
  const hide = () => { tip.style.opacity = 0; };
  const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
  const width = (el) => Math.max(280, Math.round(el.clientWidth || 560));
  const svg = (el, h) => d3.select(el).append('svg').attr('viewBox', `0 0 ${width(el)} ${h}`);
  const hover = (sel, html) => sel.on('mousemove', (e, d) => show(e, html(d))).on('mouseleave', hide);

  // legend names the series; one series needs none (the title says it).
  const legend = (el, keys) => {
    if (keys.length < 2) return;
    const box = document.createElement('div');
    box.className = 'report-legend';
    keys.forEach((k, i) => {
      const item = document.createElement('span');
      item.innerHTML = `<i style="background:${colors[i % colors.length]}"></i>${esc(k)}`;
      box.append(item);
    });
    el.append(box);
  };

  // monthTicks picks at most one month label per 70 px.
  const monthTicks = (months, W) => {
    const step = Math.max(1, Math.ceil(months.length / Math.max(1, Math.floor(W / 70))));
    return months.filter((_, i) => i % step === 0);
  };

  const draw = {};

  // bars: one horizontal bar per kind (the single row of a chartSeries).
  draw.bars = (el, d) => {
    const W = width(el);
    const rows = d.keys.map((k, i) => ({ label: k, value: d.rows[0].values[i] }));
    const h = rows.length * 28 + 8;
    const left = 120;
    const s = svg(el, h);
    const x = d3.scaleLinear([0, d3.max(rows, (r) => r.value) || 1], [left, W - 48]);
    rows.forEach((r, i) => {
      const y = 4 + i * 28;
      s.append('text').attr('class', 'lbl').attr('x', left - 8).attr('y', y + 15).attr('text-anchor', 'end').text(r.label);
      hover(s.append('rect').datum(r).attr('x', left).attr('y', y + 4).attr('height', 16).attr('rx', 2)
        .attr('width', Math.max(1, x(r.value) - left)).attr('fill', colors[0]), (v) => `${esc(v.label)}: <b>${v.value}</b>`);
      s.append('text').attr('class', 'lbl').attr('x', x(r.value) + 6).attr('y', y + 16).text(r.value);
    });
  };

  // stacked-bars: one horizontal bar per row, a segment per key.
  draw['stacked-bars'] = (el, d) => {
    const W = width(el);
    const h = d.rows.length * 34 + 32;
    const m = { l: 96, r: 16, t: 4, b: 24 };
    const s = svg(el, h);
    const rows = d.rows.map((r) => Object.fromEntries([['label', r.label], ...d.keys.map((k, i) => [k, r.values[i]])]));
    const stack = d3.stack().keys(d.keys)(rows);
    const x = d3.scaleLinear([0, d3.max(stack.at(-1), (p) => p[1]) || 1], [m.l, W - m.r]).nice();
    const y = d3.scaleBand(rows.map((r) => r.label), [m.t, h - m.b]).padding(0.3);
    stack.forEach((layer, i) => {
      hover(s.append('g').selectAll('rect').data(layer).join('rect')
        .attr('x', (p) => x(p[0]) + 1).attr('y', (p) => y(p.data.label)).attr('height', y.bandwidth()).attr('rx', 2)
        .attr('width', (p) => Math.max(0, x(p[1]) - x(p[0]) - 2)).attr('fill', colors[i % colors.length]),
      (p) => `${esc(p.data.label)} · ${esc(d.keys[i])}: <b>${p.data[d.keys[i]]}</b>`);
    });
    s.append('g').attr('class', 'axis').attr('transform', `translate(0,${h - m.b})`).call(d3.axisBottom(x).ticks(5).tickSizeOuter(0));
    s.append('g').attr('class', 'axis').attr('transform', `translate(${m.l},0)`).call(d3.axisLeft(y).tickSize(0)).select('.domain').remove();
    legend(el, d.keys);
  };

  // stacked-area: rows are months ("2026-01"), summed per key, with a hover rule.
  draw['stacked-area'] = (el, d) => {
    const W = width(el);
    const h = 220;
    const m = { l: 40, r: 12, t: 8, b: 24 };
    const s = svg(el, h);
    const parse = d3.utcParse('%Y-%m');
    const rows = d.rows.map((r) => Object.fromEntries([['t', parse(r.label)], ...d.keys.map((k, i) => [k, r.values[i]])]));
    const stack = d3.stack().keys(d.keys)(rows);
    const x = d3.scaleUtc(d3.extent(rows, (r) => r.t), [m.l, W - m.r]);
    const y = d3.scaleLinear([0, d3.max(stack.at(-1) || [], (p) => p[1]) || 1], [h - m.b, m.t]).nice();
    s.append('g').attr('class', 'axis').attr('transform', `translate(${m.l},0)`)
      .call(d3.axisLeft(y).ticks(4).tickSize(-(W - m.l - m.r))).call((g) => g.selectAll('line').attr('stroke-opacity', 0.5))
      .select('.domain').remove();
    stack.forEach((layer, i) => s.append('path').datum(layer).attr('fill', colors[i % colors.length]).attr('fill-opacity', 0.85)
      .attr('stroke', css('--bg')).attr('stroke-width', 1)
      .attr('d', d3.area().x((p) => x(p.data.t)).y0((p) => y(p[0])).y1((p) => y(p[1])).curve(d3.curveMonotoneX)));
    s.append('g').attr('class', 'axis').attr('transform', `translate(0,${h - m.b})`)
      .call(d3.axisBottom(x).tickValues(monthTicks(rows.map((r) => r.t), W)).tickFormat(d3.utcFormat('%m/%y')).tickSizeOuter(0));
    const rule = s.append('line').attr('stroke', css('--fg-3')).attr('y1', m.t).attr('y2', h - m.b).style('opacity', 0);
    s.append('rect').attr('x', m.l).attr('y', m.t).attr('width', W - m.l - m.r).attr('height', h - m.t - m.b).attr('fill', 'transparent')
      .on('mousemove', (e) => {
        const r = rows[d3.bisector((v) => v.t).center(rows, x.invert(d3.pointer(e)[0]))];
        rule.attr('x1', x(r.t)).attr('x2', x(r.t)).style('opacity', 1);
        show(e, `${d3.utcFormat('%m/%Y')(r.t)}<br>${d.keys.map((k) => `${esc(k)}: <b>${r[k]}</b>`).join('<br>')}`);
      })
      .on('mouseleave', () => { hide(); rule.style('opacity', 0); });
    legend(el, d.keys);
  };


  // fmtBytes writes bytes like the server does ("3,8 GB" in German).
  const lang = document.documentElement.lang || 'de';
  const fmtBytes = (n) => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let f = n;
    let u = 0;
    while (f >= 1000 && u < units.length - 1) { f /= 1000; u += 1; }
    return `${f.toLocaleString(lang, { maximumFractionDigits: u > 0 && f < 100 ? 1 : 0 })} ${units[u]}`;
  };

  // hatch is the fill for what Git keeps but is no longer current.
  const hatch = (s) => {
    const id = `hatch-${Math.random().toString(36).slice(2)}`;
    const p = s.append('defs').append('pattern').attr('id', id).attr('width', 6).attr('height', 6)
      .attr('patternUnits', 'userSpaceOnUse').attr('patternTransform', 'rotate(45)');
    p.append('rect').attr('width', 6).attr('height', 6).attr('fill', css('--bg-3'));
    p.append('line').attr('y2', 6).attr('stroke', css('--fg-3')).attr('stroke-width', 1.5);
    return `url(#${id})`;
  };

  // treemap: area → text, attachments, no longer current (hatched).
  draw.treemap = (el, d) => {
    const W = width(el);
    const h = 280;
    const s = svg(el, h);
    const deleted = hatch(s);
    const parts = [['attachments', colors[0]], ['text', colors[1]], ['deleted', deleted]];
    const tree = { children: d.areas.map((a) => ({ name: a.name, children: parts.map(([k]) => ({ name: d.labels[k], kind: k, area: a.name, v: a[k] })) })) };
    const root = d3.hierarchy(tree).sum((n) => n.v || 0).sort((a, b) => b.value - a.value);
    d3.treemap().size([W, h]).paddingInner(2).paddingTop(18).round(true)(root);
    (root.children || []).forEach((a) => {
      if (a.x1 - a.x0 > 60) {
        s.append('text').attr('class', 'lbl').attr('x', a.x0 + 2).attr('y', a.y0 + 13)
          .text(`${a.data.name} · ${fmtBytes(a.value)}`);
      }
    });
    hover(s.selectAll('rect.leaf').data(root.leaves().filter((l) => l.value > 0)).join('rect').attr('class', 'leaf')
      .attr('x', (l) => l.x0).attr('y', (l) => l.y0).attr('width', (l) => l.x1 - l.x0).attr('height', (l) => l.y1 - l.y0).attr('rx', 2)
      .attr('fill', (l) => parts.find(([k]) => k === l.data.kind)[1]),
    (l) => `${esc(l.data.area)} · ${esc(l.data.name)}: <b>${fmtBytes(l.value)}</b>`);
    const box = document.createElement('div');
    box.className = 'report-legend';
    box.innerHTML = `<span><i style="background:${colors[0]}"></i>${esc(d.labels.attachments)}</span>`
      + `<span><i style="background:${colors[1]}"></i>${esc(d.labels.text)}</span>`
      + `<span><i style="background:repeating-linear-gradient(45deg,${css('--fg-3')} 0 1.5px,${css('--bg-3')} 1.5px 4px)"></i>${esc(d.labels.deleted)}</span>`;
    el.append(box);
  };

  // size-lines: the repository against the current files, the gap shaded, labels at the line ends.
  draw['size-lines'] = (el, d) => {
    const W = width(el);
    const h = 220;
    const m = { l: 56, r: 96, t: 8, b: 24 };
    const s = svg(el, h);
    const parse = d3.utcParse('%Y-%m');
    const pts = d.points.map((p) => ({ ...p, t: parse(p.t) }));
    const x = d3.scaleUtc(d3.extent(pts, (p) => p.t), [m.l, W - m.r]);
    const y = d3.scaleLinear([0, d3.max(pts, (p) => p.repo) || 1], [h - m.b, m.t]).nice();
    s.append('g').attr('class', 'axis').attr('transform', `translate(${m.l},0)`)
      .call(d3.axisLeft(y).ticks(4).tickFormat(fmtBytes).tickSize(-(W - m.l - m.r)))
      .call((g) => g.selectAll('line').attr('stroke-opacity', 0.5)).select('.domain').remove();
    s.append('path').datum(pts).attr('fill', colors[0]).attr('fill-opacity', 0.12)
      .attr('d', d3.area().x((p) => x(p.t)).y0((p) => y(p.tree)).y1((p) => y(p.repo)));
    [['repo', colors[0]], ['tree', colors[1]]].forEach(([k, c]) => {
      s.append('path').datum(pts).attr('fill', 'none').attr('stroke', c).attr('stroke-width', 2)
        .attr('d', d3.line().x((p) => x(p.t)).y((p) => y(p[k])));
      s.append('text').attr('class', 'lbl').attr('x', W - m.r + 6).attr('y', y(pts.at(-1)[k]) + 4).text(d.labels[k]);
    });
    s.append('g').attr('class', 'axis').attr('transform', `translate(0,${h - m.b})`)
      .call(d3.axisBottom(x).tickValues(monthTicks(pts.map((p) => p.t), W)).tickFormat(d3.utcFormat('%m/%y')).tickSizeOuter(0));
    s.append('rect').attr('x', m.l).attr('y', m.t).attr('width', W - m.l - m.r).attr('height', h - m.t - m.b).attr('fill', 'transparent')
      .on('mousemove', (e) => {
        const p = pts[d3.bisector((v) => v.t).center(pts, x.invert(d3.pointer(e)[0]))];
        show(e, `${d3.utcFormat('%m/%Y')(p.t)}<br>${esc(d.labels.repo)}: <b>${fmtBytes(p.repo)}</b><br>${esc(d.labels.tree)}: <b>${fmtBytes(p.tree)}</b>`);
      })
      .on('mouseleave', hide);
  };

  // seq is the sequential scale for counts: from the surface step to the first chart colour.
  const seq = (max) => d3.scaleSequential([0, Math.max(1, max)], d3.interpolateRgb(css('--bg-2'), colors[0]));

  // calendar: a year of days, weeks as columns, Monday on top.
  draw.calendar = (el, d) => {
    const W = width(el);
    const parse = d3.timeParse('%Y-%m-%d');
    const days = d.days.map((x) => ({ date: parse(x.d), n: x.n }));
    const start = d3.timeMonday.floor(days[0].date);
    const weeks = d3.timeMonday.count(start, days.at(-1).date) + 1;
    const left = 24;
    const cell = Math.max(4, Math.floor((W - left) / weeks));
    const h = 7 * cell + 20;
    const s = svg(el, h);
    const c = seq(d3.max(days, (x) => x.n));
    hover(s.selectAll('rect').data(days).join('rect').attr('width', cell - 1).attr('height', cell - 1).attr('rx', 1.5)
      .attr('x', (x) => left + d3.timeMonday.count(start, x.date) * cell).attr('y', (x) => 16 + ((x.date.getDay() + 6) % 7) * cell)
      .attr('fill', (x) => c(x.n)), (x) => `${x.date.toLocaleDateString(lang)}: <b>${x.n}</b> ${esc(d.label)}`);
    d3.timeMonths(days[0].date, days.at(-1).date).forEach((m, i) => {
      if (i % (cell < 8 ? 2 : 1) === 0) {
        s.append('text').attr('class', 'lbl').attr('x', left + d3.timeMonday.count(start, m) * cell).attr('y', 10)
          .text(m.toLocaleDateString(lang, { month: 'short' }));
      }
    });
  };

  // weekhours: weekdays as rows, hours as columns.
  draw.weekhours = (el, d) => {
    const W = width(el);
    const left = 32;
    const cw = (W - left) / 24;
    const ch = Math.min(24, cw * 1.2);
    const h = 7 * ch + 20;
    const s = svg(el, h);
    const cells = d.counts.flatMap((hours, day) => hours.map((n, hour) => ({ day, hour, n })));
    const c = seq(d3.max(cells, (x) => x.n));
    hover(s.selectAll('rect').data(cells).join('rect').attr('x', (x) => left + x.hour * cw).attr('y', (x) => x.day * ch)
      .attr('width', cw - 2).attr('height', ch - 2).attr('rx', 2).attr('fill', (x) => c(x.n)),
    (x) => `${esc(d.days[x.day])}, ${x.hour}–${x.hour + 1} ${esc(d.hour)}: <b>${x.n}</b>`);
    d.days.forEach((name, i) => s.append('text').attr('class', 'lbl').attr('x', 0).attr('y', i * ch + ch / 2 + 4).text(name));
    [0, 6, 12, 18].forEach((hr) => s.append('text').attr('class', 'lbl').attr('x', left + hr * cw).attr('y', h - 4).text(`${hr} ${d.hour}`));
  };

  // age: vertical bars; bins from d.stale on are grey (old pages).
  draw.age = (el, d) => {
    const W = width(el);
    const h = 200;
    const m = { l: 32, r: 8, t: 8, b: 36 };
    const s = svg(el, h);
    const rows = d.rows.map((r, i) => ({ label: r.label, n: r.values[0], stale: i >= d.stale }));
    const x = d3.scaleBand(rows.map((r) => r.label), [m.l, W - m.r]).padding(0.2);
    const y = d3.scaleLinear([0, d3.max(rows, (r) => r.n) || 1], [h - m.b, m.t]).nice();
    s.append('g').attr('class', 'axis').attr('transform', `translate(${m.l},0)`).call(d3.axisLeft(y).ticks(4)).select('.domain').remove();
    hover(s.selectAll('rect').data(rows).join('rect').attr('x', (r) => x(r.label)).attr('width', x.bandwidth())
      .attr('y', (r) => y(r.n)).attr('height', (r) => y(0) - y(r.n)).attr('rx', 2)
      .attr('fill', (r) => (r.stale ? css('--fg-3') : colors[0])), (r) => `${esc(r.label)}: <b>${r.n}</b> ${esc(d.unit)}`);
    const axis = s.append('g').attr('class', 'axis').attr('transform', `translate(0,${h - m.b})`).call(d3.axisBottom(x).tickSize(0));
    axis.select('.domain').remove();
    if (x.bandwidth() < 64) axis.selectAll('text').attr('transform', 'rotate(-25)').attr('text-anchor', 'end');
  };

  // sunburst: space → page → subpage, each space in its colour, deeper rings lighter.
  draw.sunburst = (el, d) => {
    const W = width(el);
    const h = Math.min(320, W);
    const r = h / 2;
    const s = svg(el, h).append('g').attr('transform', `translate(${W / 2},${r})`);
    const root = d3.partition().size([2 * Math.PI, r])(d3.hierarchy(d).sum((n) => n.bytes || 0)
      .sort((a, b) => b.value - a.value));
    const arc = d3.arc().startAngle((n) => n.x0).endAngle((n) => n.x1).innerRadius((n) => n.y0)
      .outerRadius((n) => n.y1 - 2).padAngle(0.006);
    const spaceOf = (n) => { let a = n; while (a.depth > 1) a = a.parent; return a; };
    const spaces = root.children || [];
    hover(s.selectAll('path').data(root.descendants().filter((n) => n.depth && n.x1 - n.x0 > 0.002)).join('path').attr('d', arc)
      .attr('fill', (n) => colors[spaces.indexOf(spaceOf(n)) % colors.length])
      .attr('fill-opacity', (n) => Math.max(0.35, 1 - (n.depth - 1) * 0.25)),
    (n) => `${n.ancestors().reverse().slice(1).map((a) => esc(a.data.name)).join(' / ')}: <b>${fmtBytes(n.value)}</b>`);
    legend(el, spaces.map((n) => n.data.name));
  };

  // flow: new tasks above the line, done ones below, one pair per week.
  draw.flow = (el, d) => {
    const W = width(el);
    const h = 220;
    const m = { l: 32, r: 8, t: 8, b: 24 };
    const s = svg(el, h);
    const top = d3.max(d.weeks, (w) => Math.max(w.new, w.done)) || 1;
    const x = d3.scaleBand(d.weeks.map((w) => w.label), [m.l, W - m.r]).padding(0.25);
    const y = d3.scaleLinear([-top, top], [h - m.b, m.t]).nice();
    [['new', colors[0], (w) => y(w.new), (w) => y(0) - y(w.new)], ['done', colors[1], () => y(0), (w) => y(-w.done) - y(0)]]
      .forEach(([k, c, yy, hh]) => hover(s.append('g').selectAll('rect').data(d.weeks).join('rect')
        .attr('x', (w) => x(w.label)).attr('width', x.bandwidth()).attr('y', yy).attr('height', hh).attr('rx', 2).attr('fill', c),
      (w) => `${esc(w.label)}: <b>${w[k]}</b> ${esc(d.labels[k])}`));
    s.append('line').attr('x1', m.l).attr('x2', W - m.r).attr('y1', y(0)).attr('y2', y(0)).attr('stroke', css('--line'));
    s.append('g').attr('class', 'axis').attr('transform', `translate(${m.l},0)`).call(d3.axisLeft(y).ticks(4).tickFormat(Math.abs))
      .select('.domain').remove();
    const every = Math.ceil(d.weeks.length / Math.max(1, Math.floor(W / 60)));
    s.append('g').attr('class', 'axis').attr('transform', `translate(0,${h - m.b})`)
      .call(d3.axisBottom(x).tickSize(0).tickValues(d.weeks.filter((_, i) => i % every === 0).map((w) => w.label)))
      .select('.domain').remove();
    legend(el, [d.labels.new, d.labels.done]);
  };

  // quadrants: the Eisenhower fields as a 2 × 2 grid of open counts with a bar each.
  draw.quadrants = (el, d) => {
    const W = width(el);
    const gap = 8;
    const cw = (W - gap) / 2;
    const ch = 96;
    const s = svg(el, ch * 2 + gap);
    const top = d3.max(d.fields, (f) => f.open) || 1;
    d.fields.forEach((f, i) => {
      const gx = (i % 2) * (cw + gap);
      const gy = Math.floor(i / 2) * (ch + gap);
      const g = s.append('g').attr('transform', `translate(${gx},${gy})`);
      hover(g.append('rect').datum(f).attr('width', cw).attr('height', ch).attr('rx', 4).attr('fill', css('--bg-2')),
        (x) => `${esc(x.name)}: <b>${x.open}</b> ${esc(d.labels.open)} · ${x.overdue} ${esc(d.labels.overdue)} · ${x.median.toLocaleString(lang, { maximumFractionDigits: 1 })} ${esc(d.labels.median)}`);
      g.append('rect').attr('y', ch - 6).attr('width', Math.max(2, (cw * f.open) / top)).attr('height', 6).attr('rx', 2).attr('fill', colors[0]);
      g.append('text').attr('class', 'lbl').attr('x', 10).attr('y', 20).text(f.name);
      g.append('text').attr('x', 10).attr('y', 58).attr('fill', css('--fg')).style('font', `600 28px ${css('--font-mono')}`).text(f.open);
      const median = f.median ? f.median.toLocaleString(lang, { maximumFractionDigits: 1 }) : '–';
      g.append('text').attr('class', 'lbl').attr('x', 56).attr('y', 50).text(`${f.overdue} ${d.labels.overdue}`);
      g.append('text').attr('class', 'lbl').attr('x', 56).attr('y', 66).text(`${median} ${d.labels.median}`);
    });
  };

  document.querySelectorAll('.report-chart[data-chart]').forEach((el) => {
    const fn = draw[el.dataset.chart];
    if (fn && data[el.dataset.id]) fn(el, data[el.dataset.id]);
  });
})();
