package builder

import (
	"discord-music-bot/model"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

// NewSong constructs a song object from the provided song info.
// It trims and shortens the song's name, converts duration seconds to
// duration string and adds a random color.
func (builder *Builder) NewSong(info *model.SongInfo) *model.Song {
	song := new(model.Song)
	song.DurationSeconds = info.LengthSeconds
	song.DurationString = builder.secondsToTimeString(song.DurationSeconds)
	song.Name = builder.trimYoutubeSongName(info.Name)
	song.ShortName = builder.shortenYoutubeSongName(song.Name)
	song.Url = info.Url
	song.Color = rand.Intn(16777216)
	return song
}

// WrapName wraps the provided name to multiple shorter lines,
// so the full name may be displayed without widening the queue embed
func (builder *Builder) WrapName(name string) string {
	name = strings.TrimSpace(name)
	spacer := "\n> ㅤ"
	if len(name) > 100 {
		name = name[100:]
	}
	maxLength := 30
	fields := strings.Fields(name)

	fields2 := make([]string, 0)

	s := ""
	for i := 0; i <= len(fields); i++ {
		if i < len(fields) && (i == 0 || len(s+fields[i])+1 <= maxLength) {
			if len(fields[i]) > 0 {
				s += " " + fields[i]
			}
		} else {
			diff := int(math.Round((float64(maxLength) - float64(len(s))) / 3))
			if diff > 0 {
				s = strings.Repeat("\u2000", diff) + s
			}
			if len(fields2) > 0 {
				s = spacer + s
			}
			fields2 = append(fields2, s)
			if i < len(fields) {
				s = fields[i]
			}
		}
	}
	return strings.Join(fields2, "")
}

// shortenYoutubeSongName returns a substring of the provided
// song name, so that all songs in the queue appear of equal lengths.
func (builder *Builder) shortenYoutubeSongName(name string) string {
	if len(name) <= 30 {
		return name
	}
	return builder.decodeJsonEncoding(name[:30]) + "..."
}

// trimYoutubeSongName removes suffixes such as [hd], (video), [lyrics], ...
// from the youtube song name and converts it to  "Title Format"
func (builder *Builder) trimYoutubeSongName(name string) string {
	// replace [video], [film], [audio], [hd], (video), ... (text), (official), ...
	// official video, [official video], (official video), ... official spot, ...
	// lyrics, texts, ...
	r0 := regexp.MustCompile(
		`(?i)(?m)` +
			`((-\s*)?((off?ici([^(spot)]|[^(video)])*\s*` +
			`(spot|video))|(h(d|q))|(\d+p)|(\dk))$)` +
			`|` +
			`((-\s*)?(\(|\[|\|).*?(lyric(s)?|text|tekst|of(f)?ici(j)?al(ni)?|` +
			`\s*video|film|audio|spot|hd|hq|\dk)(\s*(\d+)?)(\)|\]|\|))` +
			"|" +
			`(((\+)?\s*(\()?\s*(lyric(s)?|text|tekst|(v?\s*)?(ž|z)ivo))` +
			`\s*(\))?|(#(\w+)?\s*)+$)`,
	)
	r1 := regexp.MustCompile(`((\/\/)(\/+)?)|((\|\|)(\|+)?)`)
	r2 := regexp.MustCompile(`\s*-\s*`)

	name = string(r0.ReplaceAll([]byte(name), []byte{}))
	name = string(r1.ReplaceAll([]byte(name), []byte("-")))
	name = string(r2.ReplaceAll([]byte(name), []byte(" - ")))
	name = strings.ReplaceAll(name, "`", "'")
	name = builder.decodeJsonEncoding(name)

	name = builder.toTitleString(name)
	return builder.decodeJsonEncoding(name)
}

// secondsToTimeString converts the seconds to a string
// formated as hh:mm:ss, hours and minutes are not added if zero
func (builder *Builder) secondsToTimeString(seconds int) string {
	s := ""
	hours := int(seconds / 3600)
	seconds = seconds % 3600
	minutes := int(seconds / 60)
	seconds = seconds % 60
	if hours > 0 {
		s += fmt.Sprintf("%.2d:", hours)
	}
	if minutes > 0 || hours > 0 {
		s += fmt.Sprintf("%.2d:", minutes)
	}
	return s + fmt.Sprintf("%.2d", seconds)
}

func (builder *Builder) decodeJsonEncoding(s string) string {
	name, _ := strconv.Unquote(`"` + s + `"`)
	return name
}

func (builder *Builder) toTitleString(s string) string {
	split := strings.Fields(s)
	for i, f := range split {
		f = strings.ToLower(f)
		if len(f) > 2 {
			f = strings.ToUpper(f[:1]) + f[1:]
		}
		split[i] = f
	}
	s = strings.Join(split, " ")
	s = builder.decodeJsonEncoding(s)
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	if len(s) == 0 {
		return "NoTitle"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
