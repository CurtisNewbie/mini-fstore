package hammer

import (
	"fmt"
	"os/exec"

	"github.com/curtisnewbie/miso/miso"
)

func ExtractFirstFrame(rail miso.Rail, url string, output string) error {
	cmd := exec.Command("ffmpeg", "-i", url, "-t", "1", "-frames:v", "1", "-vf", "scale=512:-2", output)
	stdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to call ffmpeg for url: %v, target output: %v, %w", url, output, err)
	}
	rail.Infof("ffmpeg finished, %v", string(stdout))
	return nil
}
