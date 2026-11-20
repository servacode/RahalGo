// Package mediascan reads text out of chat photos (OCR) and voice notes (speech-to-text) so the
// chat can catch a phone number sent as a picture or said out loud (owner decision 2026-11-20).
//
// Everything runs on our own server — nothing leaves it:
//
//	photo  tesseract (ara+eng)                     — synchronous, before the photo is delivered
//	voice  ffmpeg → 16 kHz wav → whisper.cpp (ar)  — in the background, after delivery
//
// A missing binary is not an error for the chat: Available() reports false and the caller
// delivers the media as before (dev machines and old images keep working).
package mediascan

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// Scanner reads text from a photo or a voice note.
type Scanner interface {
	// OCR returns the text read from an image file ("" if none).
	OCR(ctx context.Context, path string) (string, error)
	// Transcribe returns the speech in an audio file as text.
	Transcribe(ctx context.Context, path string) (string, error)
	// PhotoReady / VoiceReady report whether the tools are installed.
	PhotoReady() bool
	VoiceReady() bool
}

// ErrUnavailable the tool is not installed on this machine.
var ErrUnavailable = errors.New("mediascan: tool not installed")

// Exec runs the real binaries. Paths come from the environment with sane defaults:
//
//	TESSERACT_BIN  tesseract
//	FFMPEG_BIN     ffmpeg
//	WHISPER_BIN    whisper-cli
//	WHISPER_MODEL  /opt/whisper/model.bin
type Exec struct {
	Tesseract, FFmpeg, Whisper, Model string
	Threads                           string
}

// FromEnv builds the scanner from the environment.
func FromEnv() *Exec {
	get := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	return &Exec{
		Tesseract: get("TESSERACT_BIN", "tesseract"),
		FFmpeg:    get("FFMPEG_BIN", "ffmpeg"),
		Whisper:   get("WHISPER_BIN", "whisper-cli"),
		Model:     get("WHISPER_MODEL", "/opt/whisper/model.bin"),
		Threads:   get("WHISPER_THREADS", "2"),
	}
}

func have(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// PhotoReady tesseract is installed.
func (e *Exec) PhotoReady() bool { return have(e.Tesseract) }

// VoiceReady ffmpeg, whisper and its model are installed.
func (e *Exec) VoiceReady() bool {
	if !have(e.FFmpeg) || !have(e.Whisper) {
		return false
	}
	_, err := os.Stat(e.Model)
	return err == nil
}

// OCR reads the image with Arabic and English, sparse-text mode (numbers on paper, signs, screens).
func (e *Exec) OCR(ctx context.Context, path string) (string, error) {
	if !e.PhotoReady() {
		return "", ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, e.Tesseract, path, "stdout", "-l", "ara+eng", "--psm", "11")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

// Transcribe converts to 16 kHz mono wav, then runs whisper in Arabic, plain text output.
func (e *Exec) Transcribe(ctx context.Context, path string) (string, error) {
	if !e.VoiceReady() {
		return "", ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	dir, err := os.MkdirTemp("", "rg-voice-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	wav := filepath.Join(dir, "a.wav")
	if err := exec.CommandContext(ctx, e.FFmpeg, "-nostdin", "-loglevel", "error", "-i", path,
		"-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wav).Run(); err != nil {
		return "", err
	}
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, e.Whisper, "-m", e.Model, "-l", "ar", "-nt", "-np",
		"-t", e.Threads, "-f", wav)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// spoken Arabic digit words → digits (Syrian and formal spellings, with the «و» prefix).
var spoken = map[string]string{
	"صفر":  "0",
	"واحد": "1", "وحدة": "1", "واحده": "1",
	"اتنين": "2", "اثنين": "2", "إثنين": "2", "تنين": "2", "اثنان": "2",
	"تلاتة": "3", "تلاته": "3", "ثلاثة": "3", "ثلاثه": "3", "تلات": "3", "ثلاث": "3",
	"اربعة": "4", "أربعة": "4", "اربعه": "4", "أربعه": "4", "اربع": "4", "أربع": "4",
	"خمسة": "5", "خمسه": "5", "خمس": "5",
	"ستة": "6", "سته": "6", "ست": "6",
	"سبعة": "7", "سبعه": "7", "سبع": "7",
	"تمانية": "8", "تمانيه": "8", "ثمانية": "8", "ثمانيه": "8", "تمنية": "8", "تمن": "8", "ثمان": "8",
	"تسعة": "9", "تسعه": "9", "تسع": "9",
}

// SpokenDigits turns «صفر تسعة تسعة …» into «0 9 9 …» so the text guard sees a number.
func SpokenDigits(s string) string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '\n' || r == '\t' || r == '،' || r == ',' || r == '.' || r == '-'
	})
	for i, f := range fields {
		w := f
		if d, ok := spoken[w]; ok {
			fields[i] = d
			continue
		}
		if strings.HasPrefix(w, "و") {
			if d, ok := spoken[strings.TrimPrefix(w, "و")]; ok {
				fields[i] = d
			}
		}
	}
	return strings.Join(fields, " ")
}

// PhoneHits the phone numbers (and WhatsApp links) found in the text — same rules as the chat
// text guard, so prices and order numbers are not caught.
func PhoneHits(text string) []textguard.ContactHit {
	_, hits := textguard.MaskContacts(SpokenDigits(text), "***")
	out := hits[:0]
	for _, h := range hits {
		if h.Kind == "phone" || h.Kind == "link" {
			out = append(out, h)
		}
	}
	return out
}
