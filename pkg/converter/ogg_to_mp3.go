package converter

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path"

	"golang.org/x/net/context"
)

type VoiceToMP3 struct{}

func (v *VoiceToMP3) ConvertToMP3(ctx context.Context, inputPath string) (string, error) {
	slog.InfoContext(ctx, "Converting voice message to mp3...", "inputPath", inputPath)

	var (
		outputPath string
		err        error
	)
	if path.Ext(inputPath) == ".ogg" || path.Ext(inputPath) == ".oga" {
		outputPath, err = v.convertAudioToMp3(inputPath)
		if err != nil {
			return "", fmt.Errorf("converting file: %w", err)
		}
	} else {
		return "", errors.New("invalid voice message format")
	}

	slog.InfoContext(ctx, "Conversion successful", "inputPath", inputPath, "outputPath", outputPath)

	return outputPath, err
}

func (v *VoiceToMP3) convertAudioToMp3(filePath string) (string, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", fmt.Errorf("looking for `ffmpeg`: %w", err)
	}

	newFilePath := filePath + ".mp3"

	cmd := exec.Command("ffmpeg", "-i", filePath, newFilePath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return newFilePath, fmt.Errorf("running `ffmpeg`: %w", err)
	}

	return newFilePath, nil
}
