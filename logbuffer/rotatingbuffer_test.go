package logbuffer

import (
	"bytes"
	"slices"
	"testing"
)

func TestShiftSliceDown(t *testing.T) {
	t.Parallel()

	slice := []int{1, 2, 3, 4, 5}
	shiftSlice(slice)
	shouldBe := []int{2, 3, 4, 5, 5}

	if !slices.Equal(slice, shouldBe) {
		t.Fatalf("slice should have been (%v) instead it is (%v)", shouldBe, slice)
	}
}

func TestRotatingBuffer_Write(t *testing.T) {
	t.Parallel()

	buf := NewRotatingBuffer(3)

	hello, world, exam, lorum := []byte("hello"), []byte("world"), []byte("!"), []byte("lorum")

	for i, bs := range [][]byte{hello, world, exam} {
		_, _ = buf.Write(bs)

		l := buf.data[len(buf.data)-1]

		if !bytes.Equal(l, bs) {
			t.Fatalf("buffer line %v is not (%v) instead it is (%v). full string (%v)", i, string(bs), string(l), buf.String())
		}
	}

	t.Logf("full string before limit of buf+1 write: %v", buf.String())

	_, _ = buf.Write(lorum)

	if !bytes.Equal(buf.data[len(buf.data)-1], lorum) {
		t.Fatalf("last element not lorum instead: %v", buf.data[len(buf.data)-1])
	}

	if str := buf.String(); str != "world!lorum" {
		t.Fatalf("final string was not (world!lorum) instead %v", str)
	}
}

func TestRotatingBuffer_String(t *testing.T) {
	t.Parallel()

	buf := NewRotatingBuffer(3)

	hello, world, exam := []byte("hello"), []byte("world"), []byte("!")

	for _, bs := range [][]byte{hello, world, exam} {
		_, _ = buf.Write(bs)
	}

	if str := buf.String(); str != "helloworld!" {
		t.Fatalf("String() was not (helloworld!) instead it was (%v)", str)
	}
}
