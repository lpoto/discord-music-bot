package model

type Song struct {
	ID              uint   `json:"id"`               // Serial ID automatically added when the song is persisted
	Position        int    `json:"position"`         // Song's position in the queue
	Name            string `json:"name"`             // Trimmed name of the Youtube song
	ShortName       string `json:"short_name"`       // Shortened name, so that all songs' names are displayed with equal lengths
	Url             string `json:"url"`              // Url of the Youtube song
	DurationSeconds int    `json:"duration_seconds"` // Duration of the song in seconds
	DurationString  string `json:"duration_string"`  // A string representing the duration of the song in format hh:mm::ss
	Color           int    `json:"color"`            // The color of the discord embed, when this song is playing
}

type SongInfo struct {
	VideoID       string `json:"video_id"`
	Name          string `json:"name"`
	Url           string `json:"url"`
	LengthSeconds int    `json:"duration_seconds"`
}
