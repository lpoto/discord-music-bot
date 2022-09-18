package youtube

import (
	"regexp"
	"strconv"
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

func (client *YoutubeClient) extractYoutubeVideoID(url string) (string, bool) {
	content := client.getRegExpGroupValues(
		`youtube.*watch\?v=(?P<videoID>[^&\/]*)`,
		url,
		[]string{"videoID"},
	)
	v, ok := content["videoID"]
	return v, ok
}

func (client *YoutubeClient) decodeJsonEncoding(s string) string {
	name, _ := strconv.Unquote(`"` + s + `"`)
	return name
}
