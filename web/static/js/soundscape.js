// SPDX-License-Identifier: Apache-2.0
// Sounds for falling asleep, made in the browser: noise in three colours,
// waves, and a slow, soft chord with an occasional chime. Nothing is loaded.

/** A few seconds of noise to loop: white (even), pink (softer highs) or brown (deep). */
function noiseBuffer(ctx, color, seconds = 8) {
  const length = ctx.sampleRate * seconds;
  const buffer = ctx.createBuffer(2, length, ctx.sampleRate);
  for (let ch = 0; ch < 2; ch += 1) {
    const data = buffer.getChannelData(ch);
    let b0 = 0; let b1 = 0; let b2 = 0; let last = 0;
    for (let i = 0; i < length; i += 1) {
      const white = Math.random() * 2 - 1;
      if (color === 'pink') { // Paul Kellet's economy filter
        b0 = 0.99765 * b0 + white * 0.099046;
        b1 = 0.963 * b1 + white * 0.2965164;
        b2 = 0.57 * b2 + white * 1.0526913;
        data[i] = (b0 + b1 + b2 + white * 0.1848) * 0.11;
      } else if (color === 'brown') {
        last = (last + 0.02 * white) / 1.02;
        data[i] = last * 3.5;
      } else {
        data[i] = white * 0.3;
      }
    }
  }
  return buffer;
}

/** Noise as a layer; "waves" is brown noise that swells and ebbs about every twelve seconds. */
export function noise(ctx, out, kind) {
  const src = ctx.createBufferSource();
  src.buffer = noiseBuffer(ctx, kind === 'waves' ? 'brown' : kind);
  src.loop = true;
  const gain = ctx.createGain();
  src.connect(gain);
  gain.connect(out);
  const parts = [src];
  if (kind === 'waves') {
    gain.gain.value = 0.55;
    const lfo = ctx.createOscillator();
    const depth = ctx.createGain();
    lfo.frequency.value = 1 / 12;
    depth.gain.value = 0.45;
    lfo.connect(depth);
    depth.connect(gain.gain);
    lfo.start();
    parts.push(lfo);
  }
  src.start();
  return { stop: () => parts.forEach((p) => p.stop()) };
}

/** A room for the chord: an impulse response made of decaying noise. */
function reverb(ctx, seconds = 4) {
  const length = ctx.sampleRate * seconds;
  const ir = ctx.createBuffer(2, length, ctx.sampleRate);
  for (let ch = 0; ch < 2; ch += 1) {
    const data = ir.getChannelData(ch);
    for (let i = 0; i < length; i += 1) data[i] = (Math.random() * 2 - 1) * (1 - i / length) ** 3;
  }
  const conv = ctx.createConvolver();
  conv.buffer = ir;
  return conv;
}

const chord = [110, 164.81, 220, 246.94, 329.63]; // A2 E3 A3 B3 E4: open, no tension
const chimes = [440, 523.25, 587.33, 659.25, 783.99]; // A4 C5 D5 E5 G5, pentatonic

/** A soft chord whose voices breathe slowly and apart, with a chime now and then. */
export function drone(ctx, out) {
  const room = reverb(ctx);
  const tone = ctx.createBiquadFilter();
  tone.type = 'lowpass';
  tone.frequency.value = 900;
  tone.connect(room);
  room.connect(out);
  tone.connect(out);
  const parts = [];
  for (const freq of chord) {
    const osc = ctx.createOscillator();
    osc.type = 'sine';
    osc.frequency.value = freq;
    osc.detune.value = (Math.random() - 0.5) * 8;
    const amp = ctx.createGain();
    amp.gain.value = 0.05;
    const lfo = ctx.createOscillator();
    const depth = ctx.createGain();
    lfo.frequency.value = 0.03 + Math.random() * 0.05;
    depth.gain.value = 0.04;
    lfo.connect(depth);
    depth.connect(amp.gain);
    osc.connect(amp);
    amp.connect(tone);
    osc.start();
    lfo.start();
    parts.push(osc, lfo);
  }
  let timer = null;
  const chime = () => {
    const t = ctx.currentTime;
    const osc = ctx.createOscillator();
    const amp = ctx.createGain();
    osc.frequency.value = chimes[Math.floor(Math.random() * chimes.length)];
    amp.gain.setValueAtTime(0.0001, t);
    amp.gain.exponentialRampToValueAtTime(0.06, t + 0.05);
    amp.gain.exponentialRampToValueAtTime(0.0001, t + 6);
    osc.connect(amp);
    amp.connect(room);
    osc.start(t);
    osc.stop(t + 6);
    timer = setTimeout(chime, 7000 + Math.random() * 9000);
  };
  timer = setTimeout(chime, 4000);
  return { stop: () => { clearTimeout(timer); parts.forEach((p) => p.stop()); } };
}
