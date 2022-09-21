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
	// Allow only texts of max length 100
	// ... Youtube doesn't allow titles longer than 100 anyway
	if len(name) > 100 {
		name = name[100:]
	}
	// Wrap the text to lines of max length = 30
	maxLength := 30
	fields := strings.Fields(name)

	fields2 := make([]string, 0)

	// Split the text to multiple lines
	// where words are not split
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
// NOTE: Discord uses a font where the characters differ in lengths,
// but the names should still appear of equal lengths.
func (builder *Builder) shortenYoutubeSongName(name string) string {
	// TODO: currently this only shortens the name to 30 characters.
	// It should rather determine which characters are wider,...
	// and based on that define the new length of the name.
	// NOTE: Maybe canvas may be used, so the lengths are easily
	// determine based on the pixel width
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
	// TODO: should be implemented with additional patterns
	// that should be removed
	r := regexp.MustCompile(
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
	name = string(r.ReplaceAll([]byte(name), []byte{}))

	// Replace slashes and pipe lines with -
	r = regexp.MustCompile(`((\/\/)(\/+)?)|((\|\|)(\|+)?)`)
	name = string(r.ReplaceAll([]byte(name), []byte("-")))

	// Trim white space around - to a single space on each side
	r = regexp.MustCompile(`\s*-\s*`)
	name = string(r.ReplaceAll([]byte(name), []byte(" - ")))

	// Replace all ` quotes with ' so there are no code blocks
	name = strings.ReplaceAll(name, "`", "'")

	// Escape * and _ so the songs are not bold, italic or crossed
	name = strings.ReplaceAll(name, "_", `\_`)
	name = strings.ReplaceAll(name, "*", `\*`)
	name = builder.decodeJsonEncoding(name)

	// Convert the name to 'Title Format String'
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
		s += fmt.Sprintf("%d:", hours)
	}
	if hours > 0 {
		s += fmt.Sprintf("%.2d:", minutes)
	} else {
		s += fmt.Sprintf("%d:", minutes)
	}
	return s + fmt.Sprintf("%.2d", seconds)
}

func (builder *Builder) decodeJsonEncoding(s string) string {
	name, _ := strconv.Unquote(`"` + s + `"`)
	return name
}

// toTitleString converts the provided string so that
// each word is lowercase but starts with an uppercase character,
// unles the word is shorter than 3 characters, then it is
// only lowercase
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
