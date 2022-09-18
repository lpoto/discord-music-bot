package model

type Song struct {
	ID              uint      `json:"id"`
	Position        int       `json:"position"`
	Name            string    `json:"name"`
	ShortName       string    `json:"short_name"`
	Url             string    `json:"url"`
	DurationSeconds int       `json:"duration_seconds"`
	DurationString  string    `json:"duration_string"`
	Color           int       `json:"color"`
}

type SongInfo struct {
	VideoID       string `json:"video_id"`
	Name          string `json:"name"`
	Url           string `json:"url"`
	LengthSeconds int    `json:"duration_seconds"`
}
