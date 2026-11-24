# SPDX-License-Identifier: Apache-2.0
"""Speaks every line of Kanso's meditations (internal/selfcare/meditations.json) in every voice
and language, one raw WAV per line, on the AI PC (huepattl-tts-on-ai-pc skill). Run it from
ToBoldlyGlow's narration folder, which has the voices (lyra, orion) and the Qwen3-TTS helpers:

    python speak.py meditations.json out/

Writes out/<voice>/<lang>/<meditation>-<nn>.wav; lines already spoken are skipped, so an
interrupted run resumes. Loudness, trimming and MP3 happen on the Mac (encode.sh).
"""

import json
import sys
from pathlib import Path

import soundfile as sf

from synthesize import CLONE_MODEL, clone_prompt, load, speak

LANGUAGES = {"de": "German", "en": "English"}


def main(source: str, out: str) -> None:
    data = json.loads(Path(source).read_text())
    model = load(CLONE_MODEL)
    for voice in (v["id"] for v in data["voices"]):
        for code, language in LANGUAGES.items():
            prompt = clone_prompt(model, voice, code)
            folder = Path(out) / voice / code
            folder.mkdir(parents=True, exist_ok=True)
            for m in data["meditations"]:
                for i, line in enumerate(m["lines"]):
                    target = folder / f"{m['id']}-{i + 1:02d}.wav"
                    if target.exists():
                        continue
                    audio, rate = speak(model, prompt, line[code], language)
                    sf.write(target, audio, rate)
                    print(f"{target}: {len(audio) / rate:.1f} s", flush=True)


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
