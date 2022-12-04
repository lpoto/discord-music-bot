package search_test

import (
	"discord-music-bot/youtube/search"
	"testing"

	"github.com/stretchr/testify/suite"
)

type YoutubeSearchTestSuite struct {
	suite.Suite
	search *search.Search
}

// SetupSuite reuns on suite init and creates the
// search object.
func (s *YoutubeSearchTestSuite) SetupSuite() {
	s.search = search.NewSearch()
}

// TestIntegrationGetSongs gets songs by queries and urls and
// checks whether the correct number of results was returned.
func (s *YoutubeSearchTestSuite) TestIntegrationGetSongs() {
	queries := []string{
		"red hot chili peppers snow",
		"rhcp snow",
		"metallica whiskey",
		"https://www.youtube.com/watch?v=Sb5aq5HcS1A",
		"metallica witj",
		"rammstein radio",
		"https://www.youtube.com/watch?v=yuFI5KSPAt4",
	}
	songs := s.search.GetSongs(queries)
	s.Len(songs, len(queries))
}

// TestIntegrationGetSongsVerifyQueryResults gets songs by queries and
// checks the correctness of the returned results.
func (s *YoutubeSearchTestSuite) TestIntegrationGetSongsVerifyQueryResults() {
	queries := []string{
		"red hot chili peppers snow (Hey Oh)",
		"AC/DC - Thunderstruck (Official Video)",
	}

	songs := s.search.GetSongs(queries)

	s.Len(songs, len(queries))

	s.Equal(
		"Red Hot Chili Peppers - Snow (Hey Oh) (Official Music Video)",
		songs[0].Name,
	)
	s.Equal(349, songs[0].LengthSeconds)
	s.Equal("yuFI5KSPAt4", songs[0].VideoID)
	s.Equal(
		"https://www.youtube.com/watch?v=yuFI5KSPAt4",
		songs[0].Url,
	)

	s.Equal("AC/DC - Thunderstruck (Official Video)", songs[1].Name)
	s.Equal("v2AC41dglnM", songs[1].VideoID)
	s.Equal(293, songs[1].LengthSeconds)
	s.Equal(
		"https://www.youtube.com/watch?v=v2AC41dglnM",
		songs[1].Url,
	)

}

// TestIntegrationGetSongsVerifyUrlResults gets songs by urls and
// checks the correctness of the returned results.
func (s *YoutubeSearchTestSuite) TestIntegrationGetSongsVerifyUrlResults() {
	queries := []string{
		"https://www.youtube.com/watch?v=D-BhsIEzp64",
		"https://www.youtube.com/watch?v=z0NfI2NeDHI",
	}

	songs := s.search.GetSongs(queries)

	s.Len(songs, len(queries))

	s.Equal(
		"Red Hot Chili Peppers best songs",
		songs[0].Name,
	)
	s.Equal(5505, songs[0].LengthSeconds)
	s.Equal("D-BhsIEzp64", songs[0].VideoID)
	s.Equal(
		"https://www.youtube.com/watch?v=D-BhsIEzp64",
		songs[0].Url,
	)
	s.Equal("Rammstein - Radio (Official Video)", songs[1].Name)
	s.Equal("z0NfI2NeDHI", songs[1].VideoID)
	s.Equal(290, songs[1].LengthSeconds)
	s.Equal(
		"https://www.youtube.com/watch?v=z0NfI2NeDHI",
		songs[1].Url,
	)
}

// TestYoutubeSearchTestSuite runs all tests under
// the YoutubeSearchTestSuite
func TestYoutubeSearchTestSuite(t *testing.T) {
	suite.Run(t, new(YoutubeSearchTestSuite))
}
