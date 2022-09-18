package builder

import (
	"discord-music-bot/model"
	"fmt"
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

// shortenYoutubeSongName returns a substring of the provided
// song name, so that all songs in the queue appear of equal lengths.
func (builder *Builder) shortenYoutubeSongName(name string) string {
	if len(name) <= 30 {
		return name
	}
	return name[:30]
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
	return name
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
	return strings.ToUpper(s[:1]) + s[1:]
}
