package md2txt_test

import (
	"fmt"
	"testing"

	"e.coding.net/Love54dj/weizhong/etc/md2txt"
)

func TestMarkdown2Text(t *testing.T) {
	markdown := []byte("Hello, **world**!")
	expected := "Hello, world!\n"
	actual := md2txt.Markdown2Text(markdown)
	if actual != expected {
		t.Errorf("Expected: %s, Actual: %s", expected, actual)
	}
	fmt.Println(md2txt.Markdown2Text([]byte(`your test string here with **bold** and _italics_`)))
}
