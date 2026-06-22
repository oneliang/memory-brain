package profile

// MediaDetector detects media file directories
type MediaDetector struct{}

func (d *MediaDetector) Category() string {
	return CategoryMedia
}

func (d *MediaDetector) Detect(ctx *AnalysisContext) float64 {
	score := 0.0

	// File extensions (weight: 0.5)
	mediaFiles := map[string]float64{
		// Video
		".mp4":  0.6,
		".mkv":  0.6,
		".avi":  0.5,
		".mov":  0.5,
		".wmv":  0.5,
		".flv":  0.5,
		".webm": 0.5,
		// Audio
		".mp3":  0.6,
		".wav":  0.5,
		".flac": 0.5,
		".aac":  0.5,
		".ogg":  0.5,
		".m4a":  0.5,
		".wma":  0.5,
	}

	extensionScore := 0.0
	for ext, weight := range mediaFiles {
		count := ctx.GetExtensionCount(ext)
		if count > 0 {
			extensionScore += weight * min(float64(count)/3.0, 1.0) // Normalize by 3 files
		}
	}
	score += 0.5 * min(extensionScore, 1.0)

	// Directory structure (weight: 0.3)
	mediaDirs := []string{"videos", "video", "audio", "music", "recordings", "media", "movies", "songs", "podcasts"}
	dirScore := 0.0
	for _, dir := range mediaDirs {
		if ctx.HasDir(dir) {
			dirScore += 0.2
		}
	}
	score += 0.3 * min(dirScore, 1.0)

	// File names (weight: 0.2)
	mediaFileNames := []string{"video.mp4", "audio.mp3", "recording.wav", "podcast.mp3", "song.flac"}
	fileScore := 0.0
	for _, file := range mediaFileNames {
		if ctx.HasFile(file) {
			fileScore += 0.25
		}
	}
	score += 0.2 * min(fileScore, 1.0)

	return score
}
