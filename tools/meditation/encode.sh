#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Turns the raw WAVs from speak.py into the small MP3 clips Kanso serves:
# silence trimmed at both ends, loudness -16 LUFS, mono 24 kHz, VBR (speech
# about 32-40 kbit/s). Usage: tools/meditation/encode.sh <speak.py's out folder>
set -eu
src=$1
dst=web/static/audio/meditation
find "$src" -name '*.wav' | while read -r wav; do
  rel=${wav#"$src"/}
  out=$dst/${rel%.wav}.mp3
  mkdir -p "$(dirname "$out")"
  ffmpeg -nostdin -loglevel error -y -i "$wav" -af \
    "silenceremove=start_periods=1:start_threshold=-50dB,areverse,silenceremove=start_periods=1:start_threshold=-50dB,areverse,loudnorm=I=-16:TP=-1.5:LRA=11" \
    -map_metadata -1 -ac 1 -ar 24000 -c:a libmp3lame -q:a 6 "$out"
done
