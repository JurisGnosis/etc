package xunfei

import (
	"fmt"
	"os/exec"
)

func init() {
	// Check whether FFmpeg is installed
	cmd := exec.Command("ffmpeg", "-version")
	if err := cmd.Run(); err != nil {
		panic("FFmpeg is not installed or not found in PATH")
	}
}

func convertFileWavToPcm(wavFile string, pcmFile string) error {
	// Construct the FFmpeg command
	// ffmpeg -i 20250228_094921.wav -y  -acodec pcm_s16le -f s16le -ac 1 -ar 16000 20250228_094921.pcm
	command := exec.Command("ffmpeg", "-i", wavFile, "-y", "-acodec", "pcm_s16le", "-f", "s16le", "-ac", "1", "-ar", "16000", pcmFile)
	// Run the command and capture any possible error
	if err := command.Run(); err != nil {
		return fmt.Errorf("Error converting file: %v", err)
	}
	return nil
}
