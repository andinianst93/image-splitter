package config

// Config holds all CLI options for a single run.
type Config struct {
	InputPath  string  // positional argument: path to the source image
	OutputDir  string  // --output flag, default "./output"
	Rows       int     // --rows flag (required)
	Cols       int     // --cols flag (required)
	Quality    int     // --quality flag, 1-100; 0 means write PNG
	Scale      float64 // --scale flag, 1.0 means no upscaling
	AutoDetect bool    // --auto flag: detect seam positions automatically
}
