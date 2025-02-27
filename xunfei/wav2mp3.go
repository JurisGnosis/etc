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

func convertFileWavToMp3(wavFile string, mp3File string) error {
	// Construct the FFmpeg command
	command := exec.Command("ffmpeg", "-i", wavFile, mp3File)
	// Run the command and capture any possible error
	if err := command.Run(); err != nil {
		return fmt.Errorf("Error converting file: %v", err)
	}
	return nil
}
