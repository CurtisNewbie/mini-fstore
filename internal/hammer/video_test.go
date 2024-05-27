package hammer

import (
	"testing"

	"github.com/curtisnewbie/miso/miso"
)

func TestExtractFirstFrame(t *testing.T) {
	err := ExtractFirstFrame(miso.EmptyRail(), "https://curtisnewbie.com/fstore/file/stream?key=aJTTmsD14AinbtZEJNHuWmnknEYLEQ",
		"/tmp/hammer_generated.png")
	if err != nil {
		t.Fatal(err)
	}
}
