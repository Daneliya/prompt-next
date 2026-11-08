package desktop

import (
	"testing"

	"github.com/Daneliya/prompt-next/internal/library"
)

func TestEditorRequiresResultMedia(t *testing.T) {
	s := &PromptService{}
	if _, err := s.SaveOrderedImages(library.Prompt{Kind: "art"}, []library.ImageInput{{Reference: true}}); err == nil {
		t.Fatal("reference image satisfied required result image")
	}
	if _, err := s.SaveVideoWithFiles(library.Prompt{Kind: "video"}, []library.ImageInput{{Reference: true}}, 0, nil); err == nil {
		t.Fatal("reference image satisfied required result video")
	}
}
