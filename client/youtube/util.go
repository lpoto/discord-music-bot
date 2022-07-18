package youtube

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func (client *YoutubeClient) getRegExpGroupValues(reString string, s string, groups []string) map[string]string {
	re := regexp.MustCompile(reString)
	matches := re.FindAllStringSubmatch(string(s), len(groups))
	values := make(map[string]string)
	for k, g := range groups {
		for i, v := range re.SubexpNames() {
			if k < len(matches) &&
				i < len(matches[k]) &&
				v == g &&
				len(matches[k][i]) > 0 {

				values[g] = matches[k][i]
			}
		}
	}
	return values
}

func (client *YoutubeClient) secondsToTimeString(seconds int) string {
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

func (client *YoutubeClient) extractYoutubeVideoID(url string) (string, bool) {
	content := client.getRegExpGroupValues(
		`youtube.*watch\?v=(?P<videoID>[^&\/]*)`,
		url,
		[]string{"videoID"},
	)
	v, ok := content["videoID"]
	return v, ok
}

func (client *YoutubeClient) trimYoutubeSongName(name string) string {
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
	name = client.decodeJsonEncoding(name)

	words := strings.Fields(name)
	for i := range words {
		words[i] = strings.Title(strings.ToLower(words[i]))
	}
	return client.decodeJsonEncoding(strings.Join(words, " "))
}

func (client *YoutubeClient) decodeJsonEncoding(s string) string {
	name, _ := strconv.Unquote(`"` + s + `"`)
	return name
}
