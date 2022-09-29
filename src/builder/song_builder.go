package builder

import (
	"discord-music-bot/model"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/freetype/truetype"
	"github.com/sirupsen/logrus"
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
	for _, f := range fields {
		if len(f) > 60 {
			n := len(f) / 3
			fields2 = append(fields2, f[:n])
			fields2 = append(fields2, f[n:(n+n)])
			fields2 = append(fields2, f[(n+n):])
		} else if len(f) > 30 {
			n := len(f) / 2
			fields2 = append(fields2, f[:n])
			fields2 = append(fields2, f[n:])
		} else {
			fields2 = append(fields2, f)
		}
	}

	fields3 := make([]string, 0)

	// Split the text to multiple lines
	// where words are not split
	s := ""
	for i := 0; i <= len(fields2); i++ {
		if i < len(fields2) && (i == 0 || len(s+fields2[i])+1 <= maxLength) {
			if len(fields2[i]) > 0 {
				s += " " + fields2[i]
			}
		} else {
			diff := int(math.Round((float64(maxLength) - float64(len(s))) / 3))
			if diff > 0 {
				s = strings.Repeat("\u2000", diff) + s
			}
			if len(fields3) > 0 {
				s = spacer + s
			}
			fields3 = append(fields3, s)
			if i < len(fields2) {
				s = fields2[i]
			}
		}
	}
	return strings.Join(fields3, "")
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

	maxWidth := 13000
	name2 := name
	if len(name) >= 30 {
		name2 = name[:30] + "..."
	}

	// NOTE: discord uses Uni-Sans
	fontPath := "../assets/Discord-Font.ttf"
	b, err := ioutil.ReadFile(fontPath)
	if err != nil {
		logrus.Error(err)
		return name2
	}
	font, err := truetype.Parse(b)
	if err != nil {
		logrus.Error(err)
		return name2
	}
	opts := &truetype.Options{
		Size: 14, // NOTE: default font size for discord is 14
	}
	face := truetype.NewFace(font, opts)
	w := 0
	s := ""
	for _, x := range name {
		awidth, ok := face.GlyphAdvance(rune(x))
		if ok != true {
			return name[:30] + "..."
		} else {
			w2 := w + int(awidth)
			if w2 > maxWidth {
				break
			}
			w = w2
			s += string(x)
		}
	}
	if len(s) == len(name) {
		return builder.decodeJsonEncoding(s)
	}

	return builder.decodeJsonEncoding(s) + "..."
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
