package model

import (
	"math/rand"
	"time"
)

type Song struct {
	ID        uint      `json:"id"`
	Position  int       `json:"position"`
	Color     int       `json:"color"`
	Info      *SongInfo `json:"info"`
	Timestamp time.Time `json:"timestamp"`
}

type SongInfo struct {
	VideoID         string `json:"video_id"`
	Name            string `json:"name"`
	TrimmedName     string `json:"trimmed_name"`
	Url             string `json:"url"`
	DurationString  string `json:"duration_string"`
	DurationSeconds int    `json:"duration_seconds"`
}

func NewSong(info *SongInfo) *Song {
	song := new(Song)
	song.Info = info
	song.Color = rand.Intn(16777216)
	return song
}
