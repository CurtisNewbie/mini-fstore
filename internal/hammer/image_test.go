package hammer

import (
	"testing"

	"github.com/curtisnewbie/miso/miso"
)

func TestGiftCompressImage(t *testing.T) {
	in := "../test.jpg"
	out := "../compressed.jpg"

	if e := GiftCompressImage(miso.EmptyRail(), in, out); e != nil {
		t.Fatal(e)
	}
}
