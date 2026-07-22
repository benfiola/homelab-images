package webhook

type SonarrPayload struct {
	EventType string `json:"eventType"`
	Series    struct {
		Path string `json:"path"`
	} `json:"series"`
	Episodes []struct {
		ID int `json:"id"`
	} `json:"episodes"`
	EpisodeFile struct {
		Path string `json:"path"`
	} `json:"episodeFile"`
	RenamedEpisodeFiles []struct {
		PreviousPath string `json:"previousPath"`
		Path         string `json:"path"`
	} `json:"renamedEpisodeFiles"`
	DeleteReason string `json:"deleteReason"`
}

type RadarrPayload struct {
	EventType string `json:"eventType"`
	Movie     struct {
		Path string `json:"path"`
	} `json:"movie"`
	MovieFile struct {
		Path string `json:"path"`
	} `json:"movieFile"`
	RenamedMovieFiles []struct {
		PreviousPath string `json:"previousPath"`
		Path         string `json:"path"`
	} `json:"renamedMovieFiles"`
	DeleteReason string `json:"deleteReason"`
	IsUpgrade    bool   `json:"isUpgrade"`
}
