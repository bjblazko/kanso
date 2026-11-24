# Meditation clips

The guided meditations in Selfcare (`internal/selfcare/meditations.json`) are
spoken line by line. Each line is a small MP3 per voice and language under
`web/static/audio/meditation/<voice>/<lang>/<meditation>-<nn>.mp3`. The page
plays the clips with the pauses between them and makes the gong itself
(`web/static/js/meditation.js`).

The voices are Lyra and Orion. They were designed with Qwen3-TTS
(VoiceDesign), so they are not clones of real people. They live in
ToBoldlyGlow's narration folder.

## Speaking (after a change to a line's text)

1. On the AI PC (skill `huepattl-tts-on-ai-pc`): copy `speak.py` and
   `meditations.json` into ToBoldlyGlow's narration folder there, delete the
   WAVs of the changed lines, free the GPU, and run
   `python speak.py meditations.json ~/tts/jobs/kanso-meditation` detached.
2. Bring the WAVs home: `rsync -a pc:tts/jobs/kanso-meditation/ /tmp/kanso-meditation/`.
3. `tools/meditation/encode.sh /tmp/kanso-meditation`, then `go test ./internal/selfcare`
   checks that every clip is there and small.
4. Reload the chat model on the PC (see the skill).
