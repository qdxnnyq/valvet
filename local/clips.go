package main

import _ "embed"

//go:embed sounds/win.mp3
var winClip []byte

//go:embed sounds/loss.mp3
var lossClip []byte

//go:embed sounds/meow.mp3
var meowClip []byte

//go:embed sounds/knock.wav
var knockClip []byte

//go:embed sounds/bloom.wav
var bloomClip []byte

//go:embed sounds/pulse.wav
var pulseClip []byte

//go:embed sounds/clutch.wav
var clutchClip []byte
