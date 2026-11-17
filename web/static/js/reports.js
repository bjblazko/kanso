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


  document.querySelectorAll('.report-chart[data-chart]').forEach((el) => {
    const fn = draw[el.dataset.chart];
    if (fn && data[el.dataset.id]) fn(el, data[el.dataset.id]);
  });
})();
