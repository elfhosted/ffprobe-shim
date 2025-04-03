package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	ptn "github.com/middelink/go-parse-torrent-name"
)

// Path to real ffprobe (for fallback)
var REAL_FFPROBE = func() string {
	if value, exists := os.LookupEnv("REAL_FFPROBE_PATH"); exists {
		return value
	}
	return "/usr/bin/ffprobe.real" // Default value
}()

// Init logging
func init() {
	logFile, err := os.OpenFile("/tmp/ffprobe-shim.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Write directly to stderr only if logging cannot be initialized
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		os.Exit(1)
	}
	log.SetOutput(logFile)
	log.Println("Logging initialized")
	log.Printf("Using REAL_FFPROBE path: %s", REAL_FFPROBE)
}

// Pattern and template types
type PatternInfo struct {
	Pattern  string
	Template string
}

// Stream represents an ffprobe media stream
type Stream struct {
	Index              int               `json:"index"`
	CodecName          string            `json:"codec_name"`
	CodecLongName      string            `json:"codec_long_name,omitempty"`
	Profile            string            `json:"profile,omitempty"`
	CodecType          string            `json:"codec_type"`
	CodecTagString     string            `json:"codec_tag_string,omitempty"`
	CodecTag           string            `json:"codec_tag,omitempty"`
	Width              int               `json:"width,omitempty"`
	Height             int               `json:"height,omitempty"`
	CodedWidth         int               `json:"coded_width,omitempty"`
	CodedHeight        int               `json:"coded_height,omitempty"`
	ClosedCaptions     int               `json:"closed_captions,omitempty"`
	FilmGrain          int               `json:"film_grain,omitempty"`
	HasBFrames         int               `json:"has_b_frames,omitempty"`
	SampleAspectRatio  string            `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio string            `json:"display_aspect_ratio,omitempty"`
	PixFmt             string            `json:"pix_fmt,omitempty"`
	Level              int               `json:"level,omitempty"`
	ColorRange         string            `json:"color_range,omitempty"`
	ColorSpace         string            `json:"color_space,omitempty"`
	ColorTransfer      string            `json:"color_transfer,omitempty"`
	ColorPrimaries     string            `json:"color_primaries,omitempty"`
	ChromaLocation     string            `json:"chroma_location,omitempty"`
	FieldOrder         string            `json:"field_order,omitempty"`
	Refs               int               `json:"refs,omitempty"`
	IsAVC              string            `json:"is_avc,omitempty"`
	NalLengthSize      string            `json:"nal_length_size,omitempty"`
	ID                 string            `json:"id,omitempty"`
	RFrameRate         string            `json:"r_frame_rate,omitempty"`
	AvgFrameRate       string            `json:"avg_frame_rate,omitempty"`
	TimeBase           string            `json:"time_base,omitempty"`
	StartPts           int64             `json:"start_pts,omitempty"`
	StartTime          string            `json:"start_time,omitempty"`
	DurationTS         int64             `json:"duration_ts,omitempty"`
	Duration           string            `json:"duration,omitempty"`
	BitRate            string            `json:"bit_rate,omitempty"`
	BitsPerRawSample   string            `json:"bits_per_raw_sample,omitempty"`
	NbFrames           string            `json:"nb_frames,omitempty"`
	ExtradataSize      int               `json:"extradata_size,omitempty"`
	Disposition        map[string]int    `json:"disposition,omitempty"`
	Tags               map[string]string `json:"tags,omitempty"`
	SideDataList       []SideData        `json:"side_data_list,omitempty"`
	Channels           int               `json:"channels,omitempty"`       // Add this field
	SampleRate         string            `json:"sample_rate,omitempty"`    // Add this field
}

// Format represents ffprobe format information
type Format struct {
	Filename       string            `json:"filename"`
	NbStreams      int               `json:"nb_streams"`
	NbPrograms     int               `json:"nb_programs,omitempty"`
	FormatName     string            `json:"format_name"`
	FormatLongName string            `json:"format_long_name"`
	StartTime      string            `json:"start_time,omitempty"`
	Duration       string            `json:"duration,omitempty"`
	Size           string            `json:"size,omitempty"`
	BitRate        string            `json:"bit_rate,omitempty"`
	ProbeScore     int               `json:"probe_score,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
}

// Chapter represents a media chapter
type Chapter struct {
	Index int     `json:"index"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Title string  `json:"title"`
}

// SideData represents side data information
type SideData struct {
	SideDataType string `json:"side_data_type"`
	ServiceType  int    `json:"service_type,omitempty"`
}

// FFProbeResponse represents the full ffprobe output structure
type FFProbeResponse struct {
	Streams  []Stream  `json:"streams"`
	Format   Format    `json:"format"`
	Chapters []Chapter `json:"chapters,omitempty"`
}

// Define pattern matching for different file types
var PATTERNS = []PatternInfo{
	// TV Shows: ShowName.S01E02.Quality.Source.Codec.Extension
	{
		Pattern:  `.*\.S\d{2}E\d{2}.*\.(mkv|mp4|avi)`,
		Template: "tv_show",
	},
	// Movies: MovieName.Year.Quality.Source.Codec.Extension
	{
		Pattern:  `.*\.(19|20)\d{2}.*\.(mkv|mp4|avi)`,
		Template: "movie",
	},
}

// Base template responses
var TEMPLATES = map[string]FFProbeResponse{
	"tv_show": {
		Streams: []Stream{
			{
				Index:     0,
				CodecName: "h264",
				CodecType: "video",
				Width:     1920,
				Height:    1080,
				Duration:  "2700.000000",
				BitRate:   "5000000",
			},
			{
				Index:      1,
				CodecName:  "aac",
				CodecType:  "audio",
				Channels:   6,
				SampleRate: "48000",
				BitRate:    "384000",
				Duration:   "2700.000000",
			},
		},
		Format: Format{
			Filename:       "", // Will be filled in
			NbStreams:      2,
			FormatName:     "matroska,webm",
			FormatLongName: "Matroska / WebM",
			Duration:       "2700.000000",
			Size:           "1500000000",
			BitRate:        "5384000",
		},
	},
	"movie": {
		Streams: []Stream{
			{
				Index:     0,
				CodecName: "h264",
				CodecType: "video",
				Width:     1920,
				Height:    1080,
				Duration:  "7200.000000",
				BitRate:   "8000000",
			},
			{
				Index:      1,
				CodecName:  "dts",
				CodecType:  "audio",
				Channels:   6,
				Duration:   "7200.000000",
				SampleRate: "48000",
				BitRate:    "1536000",
			},
		},
		Format: Format{
			Filename:       "", // Will be filled in
			NbStreams:      2,
			FormatName:     "matroska,webm",
			FormatLongName: "Matroska / WebM",
			Duration:       "7200.000000",
			Size:           "3500000000",
			BitRate:        "9536000",
		},
	},
}

// Codec mapping
var VIDEO_CODEC_MAP = map[string]string{
	"x264":     "h264",
	"x265":     "hevc",
	"h264":     "h264",
	"hevc":     "hevc",
	"h265":     "hevc",
	"xvid":     "mpeg4",
	"divx":     "mpeg4",
	"10bit":    "hevc", // Assuming 10bit is usually HEVC
	"avc":      "h264",
	"vc-1":     "vc1",
	"bluray":   "h264", // Assuming Bluray is often h264
	"web-dl":   "h264", // Assuming web-dl is often h264
	"webrip":   "h264", // Assuming webrip is often h264
	"hdtv":     "h264", // Assuming hdtv is often h264
}

var AUDIO_CODEC_MAP = map[string]string{
	"dts":       "dts",
	"dtshd":     "dts",
	"dts-hd":    "dts",
	"truehd":    "truehd",
	"dd5.1":     "ac3",
	"dd":        "ac3",
	"ac3":       "ac3",
	"aac":       "aac",
	"eac3":      "eac3",
	"flac":      "flac",
	"atmos":     "truehd", // Assuming Atmos is often TrueHD
	"dolby":     "ac3",    // Generic Dolby is often AC3
	"5.1":       "ac3",    // Assuming 5.1 is often AC3
	"7.1":       "dts",    // Assuming 7.1 is often DTS
}

// Extract resolution and info from PTN metadata
func enhanceResponseWithPTN(response *FFProbeResponse, filepath string) {
	filename := filepath
	// Extract just the filename if it's a full path
	if strings.Contains(filepath, "/") {
		filename = filepath[strings.LastIndex(filepath, "/")+1:]
	}

	info, err := ptn.Parse(filename)
	if err != nil {
		log.Printf("Error parsing torrent name: %v", err)
		return
	}

	log.Printf("PTN info: %+v", info)

	// Set media duration based on type
	if info.Episode != 0 {
		// TV show episode - use typical episode lengths
		if strings.Contains(strings.ToLower(info.Title), "anime") {
			response.Format.Duration = "24.000000"
			for i := range response.Streams {
				if response.Streams[i].CodecType == "video" {
					response.Streams[i].Duration = "24.000000"
				} else if response.Streams[i].CodecType == "audio" {
					response.Streams[i].Duration = "24.000000" // Set audio duration
				}
			}
		} else {
			response.Format.Duration = "2700.000000"
			for i := range response.Streams {
				if response.Streams[i].CodecType == "video" {
					response.Streams[i].Duration = "2700.000000"
				} else if response.Streams[i].CodecType == "audio" {
					response.Streams[i].Duration = "2700.000000" // Set audio duration
				}
			}
		}
	} else {
		// Movie - use typical movie length
		response.Format.Duration = "7200.000000"
		for i := range response.Streams {
			if response.Streams[i].CodecType == "video" {
				response.Streams[i].Duration = "7200.000000"
			} else if response.Streams[i].CodecType == "audio" {
				response.Streams[i].Duration = "7200.000000" // Set audio duration
			}
		}
	}

	// Set resolution based on quality
	if info.Quality != "" {
		width, height := 0, 0

		if info.Quality == "720p" {
			width, height = 1280, 720
		} else if info.Quality == "1080p" {
			width, height = 1920, 1080
		} else if info.Quality == "2160p" || info.Quality == "4K" {
			width, height = 3840, 2160
		}

		if width != 0 && height != 0 {
			for i := range response.Streams {
				if response.Streams[i].CodecType == "video" {
					response.Streams[i].Width = width
					response.Streams[i].Height = height

					// Adjust bitrate based on resolution
					switch info.Quality {
					case "720p":
						response.Streams[i].BitRate = "3000000"
					case "1080p":
						response.Streams[i].BitRate = "8000000"
					case "2160p", "4K":
						response.Streams[i].BitRate = "25000000"
						response.Streams[i].CodecName = "hevc" // 4K is often HEVC
					}
				}
			}
		}
	}

	// Try to determine video codec
	videoCodec := ""
	if info.Codec != "" {
		lowerCodec := strings.ToLower(info.Codec)
		if mappedCodec, exists := VIDEO_CODEC_MAP[lowerCodec]; exists {
			videoCodec = mappedCodec
		}
	}

	// Look through other fields for codec hints
	if videoCodec == "" {
		searchFields := []string{info.Group, info.Title, info.Container}
		for _, field := range searchFields {
			lowerField := strings.ToLower(field)
			for key, value := range VIDEO_CODEC_MAP {
				if strings.Contains(lowerField, key) {
					videoCodec = value
					break
				}
			}
			if videoCodec != "" {
				break
			}
		}
	}

	// Apply video codec if found
	if videoCodec != "" {
		for i := range response.Streams {
			if response.Streams[i].CodecType == "video" {
				response.Streams[i].CodecName = videoCodec
			}
		}
	}

	// Try to determine audio codec
	audioCodec := ""
	searchFields := []string{info.Group, info.Title}
	for _, field := range searchFields {
		if field == "" {
			continue
		}
		lowerField := strings.ToLower(field)
		for key, value := range AUDIO_CODEC_MAP {
			if strings.Contains(lowerField, key) {
				audioCodec = value
				break
			}
		}
		if audioCodec != "" {
			break
		}
	}

	// Apply audio codec if found
	if audioCodec != "" {
		for i := range response.Streams {
			if response.Streams[i].CodecType == "audio" {
				response.Streams[i].CodecName = audioCodec
			}
		}
	}

	// Adjust audio channels if present
	channels := 2 // Default stereo
	if strings.Contains(info.Group, "5.1") {
		channels = 6
	} else if strings.Contains(info.Group, "7.1") {
		channels = 8
	}
	for i := range response.Streams {
		if response.Streams[i].CodecType == "audio" {
			response.Streams[i].Channels = channels
		}
	}

	// Set size based on quality and duration
	fileSize := "0"
	switch {
	case info.Quality == "720p":
		fileSize = "1000000000" // ~1GB for 720p
	case info.Quality == "1080p":
		fileSize = "3500000000" // ~3.5GB for 1080p
	case info.Quality == "2160p" || info.Quality == "4K":
		fileSize = "15000000000" // ~15GB for 4K
	default:
		fileSize = "2000000000" // Default ~2GB
	}

	response.Format.Size = fileSize

	// Set total bitrate (sum of audio and video)
	totalBitRate := 0
	for _, stream := range response.Streams {
		br, err := strconv.Atoi(stream.BitRate)
		if err == nil {
			totalBitRate += br
		}
	}

	if totalBitRate > 0 {
		response.Format.BitRate = strconv.Itoa(totalBitRate)
	}
}

// Detect which template to use based on file path
func detectFileTemplate(filepath string) string {
	filename := filepath

	// Extract just the filename if it's a full path
	if strings.Contains(filepath, "/") {
		filename = filepath[strings.LastIndex(filepath, "/")+1:]
	}

	// Try to parse with PTN first
	info, err := ptn.Parse(filename)
	if err == nil {
		if info.Episode != 0 || info.Season != 0 {
			return "tv_show"
		} else if info.Year != 0 {
			return "movie"
		}
	}

	// Fall back to regex patterns if PTN doesn't yield clear results
	for _, pattern := range PATTERNS {
		matched, err := regexp.MatchString(pattern.Pattern, filename)
		if err == nil && matched {
			return pattern.Template
		}
	}

	// If the filename contains "S01E01" format but regex didn't catch it
	if strings.Contains(strings.ToUpper(filename), "S01E") ||
		strings.Contains(strings.ToUpper(filename), "S02E") ||
		strings.Contains(strings.ToUpper(filename), "SEASON") ||
		strings.Contains(strings.ToUpper(filename), "EPISODE") {
		return "tv_show"
	}

	// If filename contains a year that looks like a movie year
	yearPattern := regexp.MustCompile(`(19|20)\d{2}`)
	if yearPattern.MatchString(filename) {
		return "movie"
	}

	return ""
}

// Generate a static ffprobe response based on template and enhance with PTN data
func generateResponse(filepath, templateName string, analyzeDuration bool) interface{} {
	template, exists := TEMPLATES[templateName]
	if !exists {
		return nil
	}

	// Deep copy by marshaling and unmarshaling
	responseBytes, err := json.Marshal(template)
	if err != nil {
		log.Printf("Error marshaling template: %v", err)
		return nil
	}

	var response FFProbeResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		log.Printf("Error unmarshaling template: %v", err)
		return nil
	}

	// Fill in filename
	response.Format.Filename = filepath

	// Extract the filename (without the path) and set it as the title
	filename := filepath[strings.LastIndex(filepath, "/")+1:]
	response.Format.Tags["title"] = filename

	// Enhance response with PTN data
	enhanceResponseWithPTN(&response, filepath)

	// Add additional fields for streams
	for i := range response.Streams {
		if response.Streams[i].CodecType == "video" {
			response.Streams[i].CodecLongName = "H.264 / AVC / MPEG-4 AVC / MPEG-4 part 10"
			response.Streams[i].Profile = "Main"
			response.Streams[i].CodecTagString = "avc1"
			response.Streams[i].CodecTag = "0x31637661"
			response.Streams[i].PixFmt = "yuv420p"
			response.Streams[i].ColorRange = "tv"
			response.Streams[i].ColorSpace = "bt709"
			response.Streams[i].ColorTransfer = "bt709"
			response.Streams[i].ColorPrimaries = "bt709"
			response.Streams[i].ChromaLocation = "left"
			response.Streams[i].FieldOrder = "progressive"
			response.Streams[i].RFrameRate = "24000/1001"
			response.Streams[i].AvgFrameRate = "24000/1001"
			response.Streams[i].TimeBase = "1/24000"
			response.Streams[i].StartTime = "0:00:00.000000"
			response.Streams[i].Duration = formatDuration(response.Streams[i].Duration)
			response.Streams[i].Disposition = map[string]int{
				"default": 1,
				"dub":     0,
				"original": 0,
				// Add other disposition fields as needed
			}
			response.Streams[i].Tags = map[string]string{
				"creation_time": "2024-10-22T13:48:39.000000Z",
				"language":      "und",
				"encoder":       "JVT/AVC Coding",
			}
		}
	}

	// Add additional fields for format
	response.Format.Tags["major_brand"] = "mp42"
	response.Format.Tags["minor_version"] = "0"
	response.Format.Tags["compatible_brands"] = "mp42isomavc1"
	response.Format.Tags["creation_time"] = "2024-10-22T13:48:39.000000Z"
	response.Format.Tags["encoder"] = "DVDFab 12.0.7.0"

	return &response
}

func formatDuration(duration string) string {
	seconds, err := strconv.ParseFloat(duration, 64)
	if err != nil {
		log.Printf("Error parsing duration: %v", err)
		return duration
	}

	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := seconds - float64(hours*3600+minutes*60)

	return fmt.Sprintf("%d:%02d:%06.3f", hours, minutes, secs)
}

// Parse ffprobe arguments to extract the file path
func parseFFProbeArgs() (string, bool, bool) {
	var inputFile string
	analyzeDuration := false
	showPixelFormats := false

	log.Println("Parsing ffprobe arguments")
	for i, arg := range os.Args {
		log.Printf("Argument %d: %s", i, arg)

		 // Detect -analyzeduration flag
		if arg == "-analyzeduration" {
			analyzeDuration = true
			log.Println("Detected -analyzeduration flag")
		}

		// Detect -show_pixel_formats flag
		if arg == "-show_pixel_formats" {
			showPixelFormats = true
			log.Println("Detected -show_pixel_formats flag")
		}

		// Look for input file (not starting with dash and exists on filesystem)
		if !strings.HasPrefix(arg, "-") {
			log.Printf("Checking if argument is a file: %s", arg)
			if fileInfo, err := os.Stat(arg); err == nil {
				if !fileInfo.IsDir() {
					inputFile = arg
					log.Printf("Detected input file: %s", inputFile)
				} else {
					log.Printf("Argument is a directory, not a file: %s", arg)
				}
			} else {
				log.Printf("File does not exist or cannot be accessed: %s, error: %v", arg, err)
			}
		}
	}

	return inputFile, analyzeDuration, showPixelFormats
}

// Execute the real ffprobe binary with the original arguments
func fallbackToRealFFProbe() {
	log.Printf("Checking if REAL_FFPROBE exists at: %s", REAL_FFPROBE)
	if _, err := os.Stat(REAL_FFPROBE); err == nil {
		log.Printf("Falling back to real ffprobe: %s %v", REAL_FFPROBE, os.Args[1:])

		cmd := exec.Command(REAL_FFPROBE, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		err := cmd.Run()
		if err != nil {
			log.Printf("Error executing real ffprobe: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	} else {
		log.Printf("Real ffprobe not found at %s. Exiting gracefully.", REAL_FFPROBE)
		os.Exit(0) // Exit successfully if REAL_FFPROBE is not found
	}
}

func main() {
    // Check if the shim should be used
    if _, useShim := os.LookupEnv("USE_FFPROBE_SHIM"); !useShim {
        log.Println("USE_FFPROBE_SHIM not set. Passing through to real ffprobe.")
        fallbackToRealFFProbe()
        return
    }

    log.Printf("FFProbe shim called with args: %s", strings.Join(os.Args, " "))

    inputFile, analyzeDuration, showPixelFormats := parseFFProbeArgs()

    // Pass -show_pixel_formats directly to the real ffprobe
    if showPixelFormats {
        log.Println("Detected -show_pixel_formats. Passing request to real ffprobe.")
        fallbackToRealFFProbe()
        return
    }

    if inputFile == "" {
        log.Printf("No input file found, falling back to real ffprobe")
        fallbackToRealFFProbe()
        return
    }

    log.Printf("Processing file: %s", inputFile)

    // Detect template to use
    templateName := detectFileTemplate(inputFile)
    log.Printf("Detected template: %s", templateName)

    if templateName == "" {
        log.Printf("No matching template for %s, falling back to real ffprobe", inputFile)
        fallbackToRealFFProbe()
        return
    }

    // Generate response
    response := generateResponse(inputFile, templateName, analyzeDuration)
    if response == nil {
        log.Printf("Failed to generate response for %s", templateName)
        fallbackToRealFFProbe()
        return
    }

    // Always output JSON
    responseJSON, err := json.MarshalIndent(response, "", "    ")
    if err != nil {
        log.Printf("Error encoding response to JSON: %v", err)
        fallbackToRealFFProbe()
        return
    }
    fmt.Print(string(responseJSON)) // Only JSON is printed to stdout
}